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
peptide_col <- params$peptide_col
protein_col <- params$protein_col
ptm_col <- ifelse(is.null(params$ptm_col), NULL, params$ptm_col)
protein_file <- ifelse(is.null(params$protein_file), NULL, params$protein_file)
protein_id_col <- ifelse(is.null(params$protein_id_col), NULL, params$protein_id_col)
log2_transform <- !is.null(params$log2_transform) && params$log2_transform != FALSE
analysis_type <- ifelse(is.null(params$analysis_type), "both", params$analysis_type)
filter_min_peptides <- ifelse(is.null(params$filter_min_peptides), 2, as.numeric(params$filter_min_peptides))
filter_min_ptm_sites <- ifelse(is.null(params$filter_min_ptm_sites), 1, as.numeric(params$filter_min_ptm_sites))
filter_min_identified <- ifelse(is.null(params$filter_min_identified), 2, as.numeric(params$filter_min_identified))
col_filter <- ifelse(is.null(params$col_filter), 0.7, as.numeric(params$col_filter))
row_filter <- ifelse(is.null(params$row_filter), 0.7, as.numeric(params$row_filter))
impute_order <- ifelse(is.null(params$impute_order), "before", params$impute_order)
impute_method <- params$impute
normalize_method <- ifelse(is.null(params$normalize_method), "center.median", params$normalize_method)
protein_aggregation <- ifelse(is.null(params$protein_aggregation), "robust", params$protein_aggregation)
robust_regression <- !is.null(params$robust_regression) && params$robust_regression != FALSE
ridge_penalty <- ifelse(is.null(params$ridge_penalty), 0, as.numeric(params$ridge_penalty))
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

message("=== MSqRob2 PTM DIA Analysis ===")
message("Input file: ", input_file)
message("Output folder: ", output_folder)
message("Annotation file: ", annotation_file)
message("Peptide column: ", peptide_col)
message("Protein column: ", protein_col)
if (!is.null(ptm_col)) message("PTM column: ", ptm_col)
message("Analysis type: ", analysis_type)
message("Log2 transform: ", log2_transform)
message("Normalization: ", normalize_method)
message("Robust regression: ", robust_regression)

message("\n[1/8] Reading peptide data...")
input_sep <- detect_delimiter(input_file)
peptide_data <- read.table(input_file, sep = input_sep, header = TRUE,
                           na.strings = c("NA", "NaN", "N/A", "#VALUE!"),
                           check.names = FALSE, stringsAsFactors = FALSE)

message("[2/8] Reading annotation...")
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

if ("BioReplicate" %in% colnames(annotation)) {
  colData_df$biorep <- annotation$BioReplicate[match(sample_cols, annotation$Sample)]
}

message("[3/8] Creating QFeatures object...")
quant_cols <- which(colnames(peptide_data) %in% sample_cols)

if (!peptide_col %in% colnames(peptide_data)) {
  stop("Peptide column '", peptide_col, "' not found in data")
}
if (!protein_col %in% colnames(peptide_data)) {
  stop("Protein column '", protein_col, "' not found in data")
}

rownames(peptide_data) <- make.unique(as.character(peptide_data[[peptide_col]]))

pe <- readQFeatures(
  peptide_data,
  quantCols = quant_cols,
  name = "peptideRaw"
)

colData(pe) <- DataFrame(colData_df)

message("Dimensions: ", nrow(pe[["peptideRaw"]]), " peptidoforms x ", ncol(pe[["peptideRaw"]]), " samples")

message("[4/8] Filtering...")

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

if (!is.null(ptm_col) && ptm_col != "" && ptm_col %in% colnames(rowData(pe[["peptideRaw"]]))) {
  keep_ptm <- !is.na(rowData(pe[["peptideRaw"]])[[ptm_col]])
  pe <- pe[keep_ptm, , ]
}

non_na_counts <- rowSums(!is.na(assay(pe[["peptideRaw"]])))
keep_features <- non_na_counts >= filter_min_identified
pe <- pe[keep_features, , ]
message("  - Removed ", sum(!keep_features), " features with fewer than ", filter_min_identified, " identifications")

peptide_counts <- table(rowData(pe[["peptideRaw"]])[[protein_col]])
proteins_to_keep <- names(peptide_counts)[peptide_counts >= filter_min_peptides]
keep_min_pep <- rowData(pe[["peptideRaw"]])[[protein_col]] %in% proteins_to_keep
pe <- pe[keep_min_pep, , ]

if (nrow(pe[["peptideRaw"]]) == 0) {
  stop("Error: No features remaining after filtering.")
}

message("After filtering: ", nrow(pe[["peptideRaw"]]), " peptidoforms for ",
        length(proteins_to_keep), " proteins")

currentAssayName <- "peptideRaw"

do_imputation <- function() {
  if (!is.null(impute_method) && impute_method != "" &&
      tolower(impute_method) != "none") {
    message(paste("  - Imputing missing values using:", impute_method))

    pe <<- impute(pe, method = impute_method, i = currentAssayName)
    currentAssayName <<- "imputedAssay"

    imputed_data <- cbind(
      rowData(pe[[currentAssayName]]),
      assay(pe[[currentAssayName]])
    )
    write.table(imputed_data,
                file = file.path(output_folder, "imputed_peptides.txt"),
                sep = "\t", row.names = FALSE, quote = FALSE)
  } else if (!is.null(impute_method) && tolower(impute_method) == "none") {
    message("  - Skipping imputation (none selected)")
  }
}

message("[5/8] Log transformation and normalization...")
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

peptidoform_intensities <- cbind(
  rowData(pe[[currentAssayName]]),
  assay(pe[[currentAssayName]])
)
write.table(peptidoform_intensities,
            file = file.path(output_folder, "peptidoform_intensities.txt"),
            sep = "\t", row.names = FALSE, quote = FALSE)

message("[6/8] Protein-level data...")

if (!is.null(protein_file) && file.exists(protein_file)) {
  message("  - Reading protein abundances from external file: ", protein_file)

  protein_sep <- detect_delimiter(protein_file)
  protein_data <- read.table(protein_file, sep = protein_sep, header = TRUE,
                             na.strings = c("NA", "NaN", "N/A", "#VALUE!"),
                             check.names = FALSE, stringsAsFactors = FALSE)

  if (is.null(protein_id_col)) {
    stop("protein_id_col must be specified when using protein_file")
  }

  if (!protein_id_col %in% colnames(protein_data)) {
    stop("Protein ID column '", protein_id_col, "' not found in protein file")
  }

  protein_sample_cols <- intersect(sample_cols, colnames(protein_data))
  if (length(protein_sample_cols) == 0) {
    stop("No matching sample columns found between peptide and protein files")
  }

  protein_quant_cols <- protein_sample_cols
  protein_matrix <- as.matrix(protein_data[, protein_quant_cols])
  rownames(protein_matrix) <- protein_data[[protein_id_col]]

  if (log2_transform) {
    message("    Applying log2 transformation to protein data")
    protein_matrix <- log2(protein_matrix)
  }

  pe <- addAssay(pe, protein_matrix, name = "protein")
  rowData(pe[["protein"]]) <- protein_data[, protein_id_col, drop = FALSE]
  colnames(rowData(pe[["protein"]])) <- protein_id_col

  pe <- msqrob(
    object = pe,
    i = "protein",
    formula = ~ 0 + condition,
    ridge = ridge_penalty,
    robust = robust_regression
  )

  message("  - Loaded ", nrow(pe[["protein"]]), " proteins from external file")

} else {
  message("  - Aggregating peptides to proteins")

  agg_fun <- switch(protein_aggregation,
    "robust" = MsCoreUtils::robustSummary,
    "sum" = colSums,
    "mean" = colMeans,
    "median" = matrixStats::colMedians,
    MsCoreUtils::robustSummary
  )

  pe <- aggregateFeatures(pe,
                         i = currentAssayName,
                         fcol = protein_col,
                         name = "protein",
                         fun = agg_fun)

  pe <- msqrob(
    object = pe,
    i = "protein",
    formula = ~ 0 + condition,
    ridge = ridge_penalty,
    robust = robust_regression
  )

  message("  - Aggregated to ", nrow(pe[["protein"]]), " proteins")
}

protein_data <- as.data.frame(rowData(pe[["protein"]]))
list_cols <- sapply(protein_data, is.list)
if (any(list_cols)) {
  protein_data <- protein_data[, !list_cols, drop = FALSE]
}
protein_intensities <- cbind(
  protein_data,
  as.data.frame(assay(pe[["protein"]]))
)
write.table(protein_intensities,
            file = file.path(output_folder, "protein_intensities.txt"),
            sep = "\t", row.names = FALSE, quote = FALSE)

message("[7/8] PTM-specific analysis...")

run_dpa <- analysis_type %in% c("DPA", "both")
run_dpu <- analysis_type %in% c("DPU", "both")

if (run_dpa) {
  message("  - Fitting models for Differential Peptidoform Abundance (DPA)...")

  pe <- msqrob(
    object = pe,
    i = currentAssayName,
    formula = ~ 0 + condition,
    ridge = ridge_penalty,
    robust = robust_regression
  )
}

if (run_dpu) {
  message("  - Computing Differential Peptidoform Usage (DPU)...")

  protein_match <- match(rowData(pe[[currentAssayName]])[[protein_col]],
                        rownames(pe[["protein"]]))

  peptide_matrix <- assay(pe[[currentAssayName]])
  protein_matrix <- assay(pe[["protein"]])[protein_match, ]

  dpu_matrix <- peptide_matrix - protein_matrix

  pe <- addAssay(pe, dpu_matrix, name = "peptideDPU")
  rowData(pe[["peptideDPU"]]) <- rowData(pe[[currentAssayName]])

  pe <- msqrob(
    object = pe,
    i = "peptideDPU",
    formula = ~ 0 + condition,
    ridge = ridge_penalty,
    robust = robust_regression
  )
}

message("[8/8] Hypothesis testing...")

all_dpa_results <- list()
all_dpu_results <- list()
all_protein_results <- list()

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

  if (run_dpa) {
    dpa_result <- hypothesisTest(
      object = pe,
      i = currentAssayName,
      contrast = L
    )

    dpa_df <- as.data.frame(rowData(dpa_result[[currentAssayName]])[[colnames(L)]])

    if (adjust_method != "BH") {
      dpa_df$adjPval <- p.adjust(dpa_df$pval, method = adjust_method)
    }

    dpa_df$comparison <- comp_label
    dpa_df$condition_A <- comparisons$condition_A[i]
    dpa_df$condition_B <- comparisons$condition_B[i]
    dpa_df$peptidoform <- rownames(dpa_df)
    dpa_df$significant <- (dpa_df$adjPval < alpha) & (abs(dpa_df$logFC) >= lfc_threshold)
    all_dpa_results[[comp_label]] <- dpa_df
  }

  if (run_dpu) {
    dpu_result <- hypothesisTest(
      object = pe,
      i = "peptideDPU",
      contrast = L
    )

    dpu_df <- as.data.frame(rowData(dpu_result[["peptideDPU"]])[[colnames(L)]])

    if (adjust_method != "BH") {
      dpu_df$adjPval <- p.adjust(dpu_df$pval, method = adjust_method)
    }

    dpu_df$comparison <- comp_label
    dpu_df$condition_A <- cond_a
    dpu_df$condition_B <- cond_b
    dpu_df$peptidoform <- rownames(dpu_df)
    dpu_df$significant <- (dpu_df$adjPval < alpha) & (abs(dpu_df$logFC) >= lfc_threshold)
    all_dpu_results[[comp_label]] <- dpu_df
  }

  protein_result <- hypothesisTest(
    object = pe,
    i = "protein",
    contrast = L
  )

  protein_df <- as.data.frame(rowData(protein_result[["protein"]])[[colnames(L)]])

  if (adjust_method != "BH") {
    protein_df$adjPval <- p.adjust(protein_df$pval, method = adjust_method)
  }

  protein_df$comparison <- comp_label
  protein_df$condition_A <- cond_a
  protein_df$condition_B <- cond_b
  protein_df$protein <- rownames(protein_df)
  protein_df$significant <- (protein_df$adjPval < alpha) & (abs(protein_df$logFC) >= lfc_threshold)
  all_protein_results[[comp_label]] <- protein_df
}

if (run_dpa && length(all_dpa_results) > 0) {
  combined_dpa <- do.call(rbind, all_dpa_results)
  combined_dpa <- combined_dpa[order(combined_dpa$pval), ]
  write.table(combined_dpa,
              file = file.path(output_folder, "dpa_results.txt"),
              sep = "\t", row.names = FALSE, quote = FALSE)

  sig_count <- sum(combined_dpa$adjPval < alpha, na.rm = TRUE)
  message("\nDPA: Found ", sig_count, " significant peptidoforms (adj. p-value < ", alpha, ")")
}

if (run_dpu && length(all_dpu_results) > 0) {
  combined_dpu <- do.call(rbind, all_dpu_results)
  combined_dpu <- combined_dpu[order(combined_dpu$pval), ]
  write.table(combined_dpu,
              file = file.path(output_folder, "dpu_results.txt"),
              sep = "\t", row.names = FALSE, quote = FALSE)

  sig_count <- sum(combined_dpu$adjPval < alpha, na.rm = TRUE)
  message("DPU: Found ", sig_count, " significant peptidoforms (adj. p-value < ", alpha, ")")
}

if (length(all_protein_results) > 0) {
  combined_protein <- do.call(rbind, all_protein_results)
  combined_protein <- combined_protein[order(combined_protein$pval), ]
  write.table(combined_protein,
              file = file.path(output_folder, "protein_results.txt"),
              sep = "\t", row.names = FALSE, quote = FALSE)

  sig_count <- sum(combined_protein$adjPval < alpha, na.rm = TRUE)
  message("Protein: Found ", sig_count, " significant proteins (adj. p-value < ", alpha, ")")
}

message("\n[QC] Generating quality control plots...")
pdf(file.path(output_folder, "qc_plots.pdf"), width = 12, height = 8)
par(mar=c(12, 4, 4, 2))

peptide_matrix <- assay(pe[[currentAssayName]])
if (ncol(peptide_matrix) > 0 && any(!is.na(peptide_matrix))) {
  boxplot(peptide_matrix, las = 2, main = "Peptidoform Intensities (Normalized)",
          ylab = "Log2 Intensity", col = rainbow(ncol(peptide_matrix)),
          cex.axis = 0.7)
} else {
  plot.new()
  text(0.5, 0.5, "No peptidoform intensity data available", cex = 1.5)
}

protein_matrix <- assay(pe[["protein"]])
if (ncol(protein_matrix) > 0 && any(!is.na(protein_matrix))) {
  boxplot(protein_matrix, las = 2, main = "Protein Intensities",
          ylab = "Log2 Intensity", col = rainbow(ncol(protein_matrix)),
          cex.axis = 0.7)
} else {
  plot.new()
  text(0.5, 0.5, "No protein intensity data available", cex = 1.5)
}

pca_data <- t(na.omit(peptide_matrix))
if (nrow(pca_data) > 2 && ncol(pca_data) > 2) {
  pca_result <- prcomp(pca_data, scale. = TRUE)
  pca_variance <- summary(pca_result)$importance[2, 1:2] * 100

  plot(pca_result$x[, 1], pca_result$x[, 2],
       col = as.numeric(colData(pe)$condition),
       pch = 19, cex = 2,
       xlab = paste0("PC1 (", round(pca_variance[1], 1), "%)"),
       ylab = paste0("PC2 (", round(pca_variance[2], 1), "%)"),
       main = "PCA - Peptidoform Level")
  legend("topright", legend = levels(colData(pe)$condition),
         col = 1:nlevels(colData(pe)$condition), pch = 19, cex = 0.8)
} else {
  plot.new()
  text(0.5, 0.5, "Insufficient data for PCA", cex = 1.5)
}

if (run_dpa && exists("combined_dpa") && nrow(combined_dpa) > 0) {
  volcano_data <- combined_dpa[!is.na(combined_dpa$logFC) & !is.na(combined_dpa$adjPval), ]
  if (nrow(volcano_data) > 0) {
    plot(volcano_data$logFC, -log10(volcano_data$adjPval),
         pch = 20, cex = 0.5,
         col = ifelse(volcano_data$adjPval < alpha, "red", "gray"),
         xlab = "Log2 Fold Change",
         ylab = "-Log10 Adjusted P-value",
         main = "Volcano Plot - DPA")
    abline(h = -log10(alpha), lty = 2, col = "blue")
    if (lfc_threshold > 0) {
      abline(v = c(-lfc_threshold, lfc_threshold), lty = 2, col = "blue")
    }
  } else {
    plot.new()
    text(0.5, 0.5, "No valid data for DPA volcano plot", cex = 1.5)
  }
} else {
  plot.new()
  text(0.5, 0.5, "No DPA results for volcano plot", cex = 1.5)
}

if (run_dpu && exists("combined_dpu") && nrow(combined_dpu) > 0) {
  volcano_data <- combined_dpu[!is.na(combined_dpu$logFC) & !is.na(combined_dpu$adjPval), ]
  if (nrow(volcano_data) > 0) {
    plot(volcano_data$logFC, -log10(volcano_data$adjPval),
         pch = 20, cex = 0.5,
         col = ifelse(volcano_data$adjPval < alpha, "red", "gray"),
         xlab = "Log2 Fold Change",
         ylab = "-Log10 Adjusted P-value",
         main = "Volcano Plot - DPU")
    abline(h = -log10(alpha), lty = 2, col = "blue")
    if (lfc_threshold > 0) {
      abline(v = c(-lfc_threshold, lfc_threshold), lty = 2, col = "blue")
    }
  } else {
    plot.new()
    text(0.5, 0.5, "No valid data for DPU volcano plot", cex = 1.5)
  }
} else {
  plot.new()
  text(0.5, 0.5, "No DPU results for volcano plot", cex = 1.5)
}

if (exists("combined_protein") && nrow(combined_protein) > 0) {
  volcano_data <- combined_protein[!is.na(combined_protein$logFC) & !is.na(combined_protein$adjPval), ]
  if (nrow(volcano_data) > 0) {
    plot(volcano_data$logFC, -log10(volcano_data$adjPval),
         pch = 20, cex = 0.5,
         col = ifelse(volcano_data$adjPval < alpha, "red", "gray"),
         xlab = "Log2 Fold Change",
         ylab = "-Log10 Adjusted P-value",
         main = "Volcano Plot - Protein")
    abline(h = -log10(alpha), lty = 2, col = "blue")
    if (lfc_threshold > 0) {
      abline(v = c(-lfc_threshold, lfc_threshold), lty = 2, col = "blue")
    }
  } else {
    plot.new()
    text(0.5, 0.5, "No valid data for protein volcano plot", cex = 1.5)
  }
} else {
  plot.new()
  text(0.5, 0.5, "No protein results for volcano plot", cex = 1.5)
}

dev.off()

message("\n=== Analysis Complete ===")
message("Results saved to: ", output_folder)
if (run_dpa) message("  - dpa_results.txt: Differential peptidoform abundance")
if (run_dpu) message("  - dpu_results.txt: Differential peptidoform usage")
message("  - protein_results.txt: Protein-level differential abundance")
message("  - peptidoform_intensities.txt: Normalized peptidoform intensities")
message("  - protein_intensities.txt: Aggregated protein intensities")
message("  - qc_plots.pdf: Quality control plots")