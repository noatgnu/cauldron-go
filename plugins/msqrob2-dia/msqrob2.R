library(QFeatures)
library(msqrob2)
library(limma)

detect_delimiter <- function(filepath) {
  ext <- tolower(tools::file_ext(filepath))
  if (ext == "csv") {
    return(",")
  } else if (ext %in% c("tsv", "txt")) {
    return("\t")
  } else {
    return("\t")
  }
}

args <- commandArgs(trailingOnly = TRUE)

parse_args <- function(args) {
  parsed <- list()
  i <- 1
  while (i <= length(args)) {
    arg <- args[i]
    if (startsWith(arg, "--")) {
      key <- substring(arg, 3)
      if (i < length(args) && !startsWith(args[i + 1], "--")) {
        value <- args[i + 1]
        parsed[[key]] <- value
        i <- i + 2
      } else {
        parsed[[key]] <- TRUE
        i <- i + 1
      }
    } else {
      i <- i + 1
    }
  }
  return(parsed)
}

params <- parse_args(args)

input_file <- params$input_file
output_folder <- params$output_folder
annotation_file <- params$annotation_file
comparison_file <- params$comparison_file
feature_id_col <- params$feature_id_col
protein_col <- params$protein_col
log2_transform <- !is.null(params$log2_transform) && params$log2_transform != FALSE
filter_min_identified <- ifelse(is.null(params$filter_min_identified), 2, as.numeric(params$filter_min_identified))
col_filter <- ifelse(is.null(params$col_filter), 0.7, as.numeric(params$col_filter))
row_filter <- ifelse(is.null(params$row_filter), 0.7, as.numeric(params$row_filter))
impute_order <- ifelse(is.null(params$impute_order), "before", params$impute_order)
impute_method <- params$impute
normalize_method <- ifelse(is.null(params$normalize_method), "center.median", params$normalize_method)
aggregation_method <- ifelse(is.null(params$aggregation_method), "robust", params$aggregation_method)
ridge_penalty <- ifelse(is.null(params$ridge_penalty), 0, as.numeric(params$ridge_penalty))
robust_regression <- !is.null(params$robust_regression) && params$robust_regression != FALSE
max_iterations <- ifelse(is.null(params$max_iterations), 20, as.numeric(params$max_iterations))
adjust_method <- ifelse(is.null(params$adjust_method), "BH", params$adjust_method)
alpha <- ifelse(is.null(params$alpha), 0.05, as.numeric(params$alpha))
lfc_threshold <- ifelse(is.null(params$lfc_threshold), 0, as.numeric(params$lfc_threshold))

if (is.null(input_file) || is.null(output_folder) || is.null(annotation_file)) {
  stop("Missing required parameters: input_file, output_folder, or annotation_file")
}

if (!dir.exists(output_folder)) {
  dir.create(output_folder, recursive = TRUE)
}

message("=== MSqRob2 DIA Analysis ===")
message("Input file: ", input_file)
message("Output folder: ", output_folder)
message("Annotation file: ", annotation_file)
message("Feature ID column: ", feature_id_col)
message("Protein column: ", protein_col)
message("Log2 transform: ", log2_transform)
message("Normalization: ", normalize_method)
message("Robust regression: ", robust_regression)

message("\n[1/7] Reading peptide data...")
input_sep <- detect_delimiter(input_file)
peptide_data <- read.table(input_file, sep = input_sep, header = TRUE,
                           na.strings = c("NA", "NaN", "N/A", "#VALUE!"),
                           check.names = FALSE, stringsAsFactors = FALSE)

message("[2/7] Reading annotation...")
annot_sep <- detect_delimiter(annotation_file)
annotation <- read.table(annotation_file, sep = annot_sep, header = TRUE,
                        stringsAsFactors = FALSE, check.names = FALSE)

if (!all(c("Sample", "Condition") %in% colnames(annotation))) {
  stop("Annotation file must contain 'Sample' and 'Condition' columns")
}

samples <- annotation$Sample
sample_cols <- intersect(samples, colnames(peptide_data))
if (length(sample_cols) == 0) {
  stop("No matching sample columns found between data and annotation")
}

message("Found ", length(sample_cols), " matching samples")

message("  - Filtering columns with >", col_filter * 100, "% missing values...")
keep_sample_cols <- c()
for (col_name in sample_cols) {
  na_count <- sum(is.na(peptide_data[[col_name]]))
  na_percentage <- na_count / nrow(peptide_data)
  if (na_percentage < col_filter) {
    keep_sample_cols <- c(keep_sample_cols, col_name)
  } else {
    message("    Removing column '", col_name, "' with ", round(na_percentage * 100, 1), "% missing values")
  }
}

if (length(keep_sample_cols) == 0) {
  stop("No sample columns remaining after filtering for missing values")
}

sample_cols <- keep_sample_cols
message("  - ", length(sample_cols), " samples remaining after column filtering")

colData_df <- data.frame(
  sample = sample_cols,
  condition = factor(make.names(annotation$Condition[match(sample_cols, annotation$Sample)])),
  row.names = sample_cols
)

message("[3/7] Creating QFeatures object...")
quant_cols <- which(colnames(peptide_data) %in% sample_cols)

if (!feature_id_col %in% colnames(peptide_data)) {
  stop("Feature ID column '", feature_id_col, "' not found in data")
}
if (!protein_col %in% colnames(peptide_data)) {
  stop("Protein column '", protein_col, "' not found in data")
}

rownames(peptide_data) <- make.unique(as.character(peptide_data[[feature_id_col]]))

pe <- readQFeatures(
  peptide_data,
  quantCols = quant_cols,
  name = "peptideRaw"
)

colData(pe) <- DataFrame(colData_df)

message("Dimensions: ", nrow(pe[["peptideRaw"]]), " features x ", ncol(pe[["peptideRaw"]]), " samples")

message("[4/7] Filtering...")
message("  - Converting zeros to NA and filtering rows with >", row_filter * 100, "% missing values...")
pe <- zeroIsNA(pe, i = seq_along(pe))
initial_features <- nrow(pe[["peptideRaw"]])
pe <- filterNA(pe, i = seq_along(pe), pNA = row_filter)
filtered_features <- initial_features - nrow(pe[["peptideRaw"]])
if (filtered_features > 0) {
  message("    Removed ", filtered_features, " features with >", row_filter * 100, "% missing values")
}

keep_protein <- !is.na(rowData(pe[["peptideRaw"]])[[protein_col]])
pe <- pe[keep_protein, , ]

non_na_counts <- rowSums(!is.na(assay(pe[["peptideRaw"]])))
keep_features <- non_na_counts >= filter_min_identified
pe <- pe[keep_features, , ]
message("  - Removed ", sum(!keep_features), " features with fewer than ", filter_min_identified, " identifications")

if (nrow(pe[["peptideRaw"]]) == 0) {
  stop("Error: No features remaining after filtering.")
}

currentAssayName <- "peptideRaw"

do_imputation <- function() {
  if (!is.null(impute_method) && impute_method != "" &&
      tolower(impute_method) != "none") {
    message(paste("  - Imputing missing values using:", impute_method))
    pe <<- impute(pe, method = impute_method, i = currentAssayName, name = "peptideImputed")
    currentAssayName <<- "peptideImputed"

    imputed_data <- cbind(
      as.data.frame(rowData(pe[[currentAssayName]])),
      as.data.frame(assay(pe[[currentAssayName]]))
    )
    list_cols <- sapply(imputed_data, is.list)
    if (any(list_cols)) {
      imputed_data <- imputed_data[, !list_cols, drop = FALSE]
    }
    write.table(imputed_data,
                file = file.path(output_folder, "imputed_peptides.txt"),
                sep = "\t", row.names = FALSE, quote = FALSE)
  }
}

message("[5/7] Log transformation and normalization...")
if (log2_transform) {
  message("  - Applying log2 transformation")
  pe <- logTransform(pe, base = 2, i = currentAssayName, name = "peptideLog")
  currentAssayName <- "peptideLog"
} else {
  pe <- addAssay(pe, assay(pe[[currentAssayName]]), name = "peptideLog")
  currentAssayName <- "peptideLog"
}

if (impute_order == "before") {
  do_imputation()
}

if (normalize_method != "none") {
  message("  - Normalizing with method: ", normalize_method)
  pe <- normalize(pe, i = currentAssayName, name = "peptideNorm", method = normalize_method)
  currentAssayName <- "peptideNorm"
} else {
  pe <- addAssay(pe, assay(pe[[currentAssayName]]), name = "peptideNorm")
  currentAssayName <- "peptideNorm"
}

if (impute_order == "after") {
  do_imputation()
}

normalized_peptide_data <- cbind(
  as.data.frame(rowData(pe[[currentAssayName]])),
  as.data.frame(assay(pe[[currentAssayName]]))
)
list_cols <- sapply(normalized_peptide_data, is.list)
if (any(list_cols)) {
  normalized_peptide_data <- normalized_peptide_data[, !list_cols, drop = FALSE]
}

write.table(normalized_peptide_data,
            file = file.path(output_folder, "normalized_peptides.txt"),
            sep = "\t", row.names = FALSE, quote = FALSE)

message("[6/7] Protein aggregation and model fitting...")
agg_fun <- switch(aggregation_method,
  "robust" = MsCoreUtils::robustSummary,
  "sum" = colSums,
  "mean" = colMeans,
  "median" = matrixStats::colMedians,
  "iPQF" = MsCoreUtils::robustSummary,
  MsCoreUtils::robustSummary
)

pe <- aggregateFeatures(pe,
                       i = currentAssayName,
                       fcol = protein_col,
                       name = "protein",
                       fun = agg_fun)

message("  - Aggregated to ", nrow(pe[["protein"]]), " proteins")

formula <- ~ 0 + condition

pe <- msqrob(
  object = pe,
  i = "protein",
  formula = formula,
  ridge = ridge_penalty,
  robust = robust_regression,
  maxitRob = max_iterations
)

protein_intensities <- cbind(
  as.data.frame(rowData(pe[["protein"]])),
  as.data.frame(assay(pe[["protein"]]))
)
list_cols <- sapply(protein_intensities, is.list)
if (any(list_cols)) {
  protein_intensities <- protein_intensities[, !list_cols, drop = FALSE]
}

write.table(protein_intensities,
            file = file.path(output_folder, "protein_intensities.txt"),
            sep = "\t", row.names = FALSE, quote = FALSE)

message("[7/7] Hypothesis testing...")

all_results <- list()

if (!is.null(comparison_file) && file.exists(comparison_file)) {
  comp_sep <- detect_delimiter(comparison_file)
  comparisons <- read.table(comparison_file, sep = comp_sep, header = TRUE,
                           stringsAsFactors = FALSE, check.names = FALSE)
} else {
  message("  - Performing all pairwise comparisons")
  conditions <- levels(colData(pe)$condition)
  if (length(conditions) < 2) {
    stop("Need at least 2 conditions for comparisons")
  }

  comparisons <- data.frame(
    comparison_label = character(),
    condition_A = character(),
    condition_B = character(),
    stringsAsFactors = FALSE
  )

  for (i in 1:(length(conditions) - 1)) {
    for (j in (i + 1):length(conditions)) {
      comparisons <- rbind(comparisons, data.frame(
        comparison_label = paste0(conditions[j], "_vs_", conditions[i]),
        condition_A = conditions[i],
        condition_B = conditions[j],
        stringsAsFactors = FALSE
      ))
    }
  }
}

for (i in 1:nrow(comparisons)) {
  comp_label <- comparisons$comparison_label[i]
  cond_a <- make.names(comparisons$condition_A[i])
  cond_b <- make.names(comparisons$condition_B[i])

  message("  - Testing: ", comp_label, " (", cond_b, " vs ", cond_a, ")")

  contrast_str <- paste0("condition", cond_b, " - condition", cond_a, " = 0")
  L <- makeContrast(contrast_str, parameterNames = paste0("condition", levels(colData(pe)$condition)))

  result <- hypothesisTest(object = pe, i = "protein", contrast = L)
  result_df <- as.data.frame(rowData(result[["protein"]])[[colnames(L)]])

  if (adjust_method != "BH") {
    result_df$adjPval <- p.adjust(result_df$pval, method = adjust_method)
  }

  result_df$comparison <- comp_label
  result_df$condition_A <- comparisons$condition_A[i]
  result_df$condition_B <- comparisons$condition_B[i]
  result_df$protein <- rownames(result_df)
  result_df$significant <- (result_df$adjPval < alpha) & (abs(result_df$logFC) >= lfc_threshold)
  all_results[[comp_label]] <- result_df
}

combined_results <- do.call(rbind, all_results)
combined_results <- combined_results[order(combined_results$pval), ]
write.table(combined_results,
            file = file.path(output_folder, "msqrob2_results.txt"),
            sep = "\t", row.names = FALSE, quote = FALSE)

sig_count <- sum(combined_results$adjPval < alpha, na.rm = TRUE)
message("\nFound ", sig_count, " significant proteins (adj. p-value < ", alpha, ")")

message("\n[QC] Generating quality control plots...")
pdf(file.path(output_folder, "qc_plot.pdf"), width = 12, height = 8)
par(mar=c(12, 4, 4, 2))

intensity_matrix <- assay(pe[[currentAssayName]])
if (ncol(intensity_matrix) > 0 && any(!is.na(intensity_matrix))) {
  boxplot(intensity_matrix, las = 2, main = "Peptide Intensities (Normalized)",
          ylab = "Log2 Intensity", col = rainbow(ncol(intensity_matrix)),
          cex.axis = 0.7)
}

pca_data <- t(na.omit(intensity_matrix))
if (nrow(pca_data) > 2 && ncol(pca_data) > 2) {
  pca_result <- prcomp(pca_data, scale. = TRUE)
  pca_variance <- summary(pca_result)$importance[2, 1:2] * 100
  plot(pca_result$x[, 1], pca_result$x[, 2], col = as.numeric(colData(pe)$condition),
       pch = 19, cex = 2, xlab = paste0("PC1 (", round(pca_variance[1], 1), "%)"),
       ylab = paste0("PC2 (", round(pca_variance[2], 1), "%)"), main = "PCA - Peptide Level")
  legend("topright", legend = levels(colData(pe)$condition), col = 1:nlevels(colData(pe)$condition), pch = 19, cex = 0.8)
}

protein_matrix <- assay(pe[["protein"]])
if (ncol(protein_matrix) > 0 && any(!is.na(protein_matrix))) {
  boxplot(protein_matrix, las = 2, main = "Protein Intensities (Aggregated)",
          ylab = "Log2 Intensity", col = rainbow(ncol(protein_matrix)),
          cex.axis = 0.7)
}

if (nrow(combined_results) > 0) {
  volcano_data <- combined_results[!is.na(combined_results$logFC) & !is.na(combined_results$adjPval), ]
  if (nrow(volcano_data) > 0) {
    plot(volcano_data$logFC, -log10(volcano_data$adjPval), pch = 20, cex = 0.5,
         col = ifelse(volcano_data$adjPval < alpha, "red", "gray"),
         xlab = "Log2 Fold Change", ylab = "-Log10 Adjusted P-value", main = "Volcano Plot")
    abline(h = -log10(alpha), lty = 2, col = "blue")
    if (lfc_threshold > 0) {
      abline(v = c(-lfc_threshold, lfc_threshold), lty = 2, col = "blue")
    }
  }
}
dev.off()

message("\n=== Analysis Complete ===")
message("Results saved to: ", output_folder)
message("  - msqrob2_results.txt: Differential expression analysis results")
message("  - protein_intensities.txt: Aggregated protein intensities")
message("  - normalized_peptides.txt: Normalized peptide-level intensities")
if (any(grepl("peptideImputed", names(pe)))) message("  - imputed_peptides.txt: Imputed peptide intensities")
message("  - qc_plot.pdf: Quality control plots")
