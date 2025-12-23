library(QFeatures)
library(msqrob2)
library(limma)
library(MsCoreUtils)

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
filter_min_peptides <- ifelse(is.null(params$filter_min_peptides), 2, as.numeric(params$filter_min_peptides))
remove_shared_peptides <- !is.null(params$remove_shared_peptides) && params$remove_shared_peptides != FALSE
col_filter <- ifelse(is.null(params$col_filter), 0.7, as.numeric(params$col_filter))
row_filter <- ifelse(is.null(params$row_filter), 0.7, as.numeric(params$row_filter))
impute_order <- ifelse(is.null(params$impute_order), "after", params$impute_order)
impute_method <- params$impute
normalize_method <- ifelse(is.null(params$normalize_method), "center.median", params$normalize_method)
aggregation_order <- ifelse(is.null(params$aggregation_order), "after", params$aggregation_order)
summarization_method <- ifelse(is.null(params$summarization_method), "robust", params$summarization_method)
model_run_effect <- !is.null(params$model_run_effect) && params$model_run_effect != FALSE
robust_regression <- !is.null(params$robust_regression) && params$robust_regression != FALSE
ridge_penalty <- ifelse(is.null(params$ridge_penalty), 0, as.numeric(params$ridge_penalty))
max_iterations <- ifelse(is.null(params$max_iterations), 20, as.numeric(params$max_iterations))
adjust_method <- ifelse(is.null(params$adjust_method), "BH", params$adjust_method)
alpha <- ifelse(is.null(params$alpha), 0.05, as.numeric(params$alpha))
lfc_threshold <- ifelse(is.null(params$lfc_threshold), 0, as.numeric(params$lfc_threshold))
exclude_conditions <- params$exclude_conditions
remove_norm_channel <- !is.null(params$remove_norm_channel) && params$remove_norm_channel != FALSE

if (remove_norm_channel) {
  if (is.null(exclude_conditions) || exclude_conditions == "") {
    exclude_conditions <- "Norm"
  } else {
    exclude_conditions <- paste(exclude_conditions, "Norm", sep = ",")
  }
}

if (is.null(input_file) || is.null(output_folder) || is.null(annotation_file)) {
  stop("Missing required parameters: input_file, output_folder, or annotation_file")
}

if (!dir.exists(output_folder)) {
  dir.create(output_folder, recursive = TRUE)
}

message("=== MSqRob2 MaxQuant TMT Analysis ===")

message("\n[1/7] Reading data...")
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

colnames(peptide_data) <- make.names(colnames(peptide_data))
annotation$Sample <- make.names(annotation$Sample)
feature_id_col <- make.names(feature_id_col)
protein_col <- make.names(protein_col)

samples <- annotation$Sample
sample_cols_in_data <- intersect(samples, colnames(peptide_data))
if (length(sample_cols_in_data) == 0) {
  message("ERROR: No matching sample columns found")
  message("  Annotation samples: ", paste(head(samples, 10), collapse = ", "))
  message("  Data columns (first 20): ", paste(head(colnames(peptide_data), 20), collapse = ", "))
  stop("No matching sample columns found between data and annotation (checked sanitized names)")
}

message("Found ", length(sample_cols_in_data), " matching samples")

message("  - Technical filtering: columns with >", col_filter * 100, "% missing values...")
keep_sample_cols <- c()
for (col_name in sample_cols_in_data) {
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

all_good_sample_cols <- keep_sample_cols
message("  - ", length(all_good_sample_cols), " samples retained for initial processing")

message("[3/7] Creating QFeatures object...")

peptide_data_subset <- peptide_data
# Create unique feature ID by combining sequence and protein
unique_ids <- paste(peptide_data_subset[[feature_id_col]], peptide_data_subset[[protein_col]], sep = "_")
rownames(peptide_data_subset) <- make.unique(as.character(unique_ids))

quant_cols <- which(colnames(peptide_data_subset) %in% all_good_sample_cols)
pe <- readQFeatures(
  peptide_data_subset,
  quantCols = quant_cols,
  name = "peptideRaw"
)

colData_df_all <- data.frame(
  sample = all_good_sample_cols,
  condition = factor(make.names(annotation$Condition[match(all_good_sample_cols, annotation$Sample)])),
  row.names = all_good_sample_cols
)

if ("BioReplicate" %in% colnames(annotation)) {
  colData_df_all$biorep <- factor(make.names(annotation$BioReplicate[match(all_good_sample_cols, annotation$Sample)]))
}

if ("Channel" %in% colnames(annotation)) {
  colData_df_all$channel <- annotation$Channel[match(all_good_sample_cols, annotation$Sample)]
}

if ("Run" %in% colnames(annotation)) {
  colData_df_all$run <- factor(annotation$Run[match(all_good_sample_cols, annotation$Sample)])
}

colData(pe) <- DataFrame(colData_df_all)
pe <- pe[, rownames(colData_df_all)]

message("Dimensions: ", nrow(pe[["peptideRaw"]]), " peptides x ", ncol(pe[["peptideRaw"]]), " samples")

message("[4/7] Filtering rows...")
pe <- zeroIsNA(pe, i = "peptideRaw")
initial_features <- nrow(pe[["peptideRaw"]])
pe <- filterNA(pe, i = "peptideRaw", pNA = row_filter)
filtered_features <- initial_features - nrow(pe[["peptideRaw"]])
if (filtered_features > 0) {
  message("    Removed ", filtered_features, " features with >", row_filter * 100, "% missing values")
}

keep_protein <- !is.na(rowData(pe[["peptideRaw"]])[[protein_col]]) & (rowData(pe[["peptideRaw"]])[[protein_col]] != "")
pe <- pe[keep_protein, , ]

if (remove_shared_peptides) {
  message("  - Removing shared peptides...")
  protein_counts <- rowData(pe[["peptideRaw"]])[[protein_col]]
  shared_peptides <- grepl(";", protein_counts) | grepl(",", protein_counts)
  pe <- pe[!shared_peptides, , ]
  message("    Removed ", sum(shared_peptides), " shared peptides")
}

currentAssayName <- "peptideRaw"

do_imputation <- function() {
  if (!is.null(impute_method) && impute_method != "" &&
      tolower(impute_method) != "none") {
    message(paste("  - Imputing missing values using:", impute_method))
    pe <<- impute(pe, method = impute_method, i = currentAssayName, name = "peptideImputed")
    currentAssayName <<- "peptideImputed"

    imputed_data <- cbind(
      rowData(pe[[currentAssayName]]),
      assay(pe[[currentAssayName]])
    )
    list_cols <- sapply(imputed_data, is.list)
    if (any(list_cols)) { imputed_data <- imputed_data[, !list_cols, drop = FALSE] }
    write.table(imputed_data, file = file.path(output_folder, "imputed_peptides.txt"), sep = "\t", row.names = FALSE, quote = FALSE)
  }
}

message("[5/7] Log transformation...")
if (log2_transform) {
  message("  - Applying log2 transformation")
  pe <- logTransform(pe, base = 2, i = currentAssayName, name = "peptideLog")
  currentAssayName <- "peptideLog"
} else {
  pe <- addAssay(pe, assay(pe[[currentAssayName]]), name = "peptideLog")
  currentAssayName <- "peptideLog"
}

if (!is.null(exclude_conditions) && exclude_conditions != "") {
  excluded <- trimws(strsplit(exclude_conditions, ",")[[1]])
  samples_to_keep <- rownames(colData(pe))[!(colData(pe)$condition %in% make.names(excluded))]
  if (length(samples_to_keep) == 0) {
    stop("No samples remaining after excluding conditions: ", exclude_conditions)
  }
  pe <- pe[, samples_to_keep]
  colData(pe)$condition <- droplevels(colData(pe)$condition)
  message("  - Excluded conditions: ", paste(excluded, collapse=", "), ". Samples remaining for analysis: ", length(samples_to_keep))
}

do_normalization <- function() {
  if (normalize_method != "none") {
    message("  - Normalizing: ", normalize_method)
    pe <<- normalize(pe, i = currentAssayName, name = "peptideNorm", method = normalize_method)
    currentAssayName <<- "peptideNorm"
  } else {
    pe <<- addAssay(pe, assay(pe[[currentAssayName]]), name = "peptideNorm")
    currentAssayName <<- "peptideNorm"
  }
}

do_aggregation <- function() {
  peptide_counts <- table(rowData(pe[[currentAssayName]])[[protein_col]])
  proteins_to_keep <- names(peptide_counts)[peptide_counts >= filter_min_peptides]
  keep_min_pep <- rowData(pe[[currentAssayName]])[[protein_col]] %in% proteins_to_keep
  pe <<- pe[keep_min_pep, , ]

  if (nrow(pe[[currentAssayName]]) == 0) {
    stop("Error: No features remaining after protein-level filtering.")
  }

  message("  - Aggregating: ", summarization_method, " (", nrow(pe[[currentAssayName]]), " peptides for ", length(proteins_to_keep), " proteins)")

  # Save normalized peptide data before aggregation
  normalized_peptides_out <- cbind(
    as.data.frame(rowData(pe[[currentAssayName]])),
    as.data.frame(assay(pe[[currentAssayName]]))
  )
  list_cols <- sapply(normalized_peptides_out, is.list)
  if (any(list_cols)) { normalized_peptides_out <- normalized_peptides_out[, !list_cols, drop = FALSE] }
  write.table(normalized_peptides_out, file = file.path(output_folder, "normalized_peptides.txt"), sep = "\t", row.names = FALSE, quote = FALSE)

  summ_fun <- switch(summarization_method,
    "robust" = MsCoreUtils::robustSummary,
    "sum" = colSums,
    "mean" = colMeans,
    "median" = matrixStats::colMedians,
    MsCoreUtils::robustSummary
  )

  pe <<- pe[, colnames(pe[[currentAssayName]])]
  pe <<- aggregateFeatures(pe, i = currentAssayName, fcol = protein_col, name = "protein", fun = summ_fun)
  currentAssayName <<- "protein"
  message("  - Result: ", nrow(pe[["protein"]]), " proteins")
}

message("[6/7] Processing: aggregation ", aggregation_order, " normalization...")

if (aggregation_order == "before") {
  do_imputation()
  do_aggregation()
  do_normalization()
} else {
  if (impute_order == "before") {
    do_imputation()
  }
  do_normalization()
  if (impute_order == "after") {
    do_imputation()
  }
  do_aggregation()
}

message("[6.5/7] Model fitting...")

# Check if run effect is possible (need >1 run)
use_run_effect <- FALSE
if (model_run_effect && "run" %in% colnames(colData(pe))) {
  n_runs <- length(unique(colData(pe)$run))
  if (n_runs > 1) {
    use_run_effect <- TRUE
    formula <- ~ 0 + condition + (1|run)
    message("  - Using formula with run effect (", n_runs, " runs): ", deparse(formula))
  } else {
    message("  - Warning: model_run_effect=TRUE but only 1 run detected. Using fixed effects only.")
    formula <- ~ 0 + condition
  }
} else {
  formula <- ~ 0 + condition
}

pe <- msqrob(
  object = pe,
  i = currentAssayName,
  formula = formula,
  ridge = ridge_penalty,
  robust = robust_regression,
  maxitRob = max_iterations
)

# Check if model fitting succeeded
if (!"msqrobModels" %in% colnames(rowData(pe[[currentAssayName]]))) {
  stop("Model fitting failed. Check if there are enough observations per condition.")
}

protein_intensities <- cbind(
  as.data.frame(rowData(pe[["protein"]])),
  as.data.frame(assay(pe[["protein"]]))
)
list_cols <- sapply(protein_intensities, is.list)
if (any(list_cols)) { protein_intensities <- protein_intensities[, !list_cols, drop = FALSE] }
write.table(protein_intensities, file = file.path(output_folder, "protein_intensities.txt"), sep = "\t", row.names = FALSE, quote = FALSE)

message("[7/7] Hypothesis testing...")

if (!is.null(comparison_file) && file.exists(comparison_file)) {
  comp_sep <- detect_delimiter(comparison_file)
  comparisons <- read.table(comparison_file, sep = comp_sep, header = TRUE,
                           stringsAsFactors = FALSE, check.names = FALSE)
  message("  - Loaded ", nrow(comparisons), " comparisons from file")
} else {
  message("  - Performing all pairwise comparisons")
  conditions_levels <- levels(colData(pe)$condition)
  message("  - Available conditions: ", paste(conditions_levels, collapse = ", "))

  if (length(conditions_levels) < 2) {
    stop("Need at least 2 conditions for comparisons. Found: ", length(conditions_levels))
  }

  comparisons <- data.frame(
    comparison_label = character(),
    condition_A = character(),
    condition_B = character(),
    stringsAsFactors = FALSE
  )

  for (i in 1:(length(conditions_levels) - 1)) {
    for (j in (i + 1):length(conditions_levels)) {
      comparisons <- rbind(comparisons, data.frame(
        comparison_label = paste0(conditions_levels[j], "_vs_", conditions_levels[i]),
        condition_A = conditions_levels[i],
        condition_B = conditions_levels[j],
        stringsAsFactors = FALSE
      ))
    }
  }
  message("  - Generated ", nrow(comparisons), " pairwise comparisons")
}

if (nrow(comparisons) == 0) {
  stop("No comparisons to test")
}

all_results <- list()
# Valid parameter names for makeContrast
valid_params <- paste0("condition", levels(colData(pe)$condition))

# Filter out proteins with NULL models before hypothesis testing
protein_models <- rowData(pe[["protein"]])$msqrobModels
valid_models <- !sapply(protein_models, is.null)
if (sum(valid_models) == 0) {
  stop("No valid protein models for hypothesis testing. All proteins failed to fit.")
}
if (sum(!valid_models) > 0) {
  message("  - Filtering ", sum(!valid_models), " proteins with failed models")
  # Filter the protein assay only
  protein_assay_filtered <- pe[["protein"]][valid_models, ]
  # Replace protein assay with filtered version
  pe <- removeAssay(pe, "protein")
  pe <- addAssay(pe, protein_assay_filtered, name = "protein")
}

for (i in 1:nrow(comparisons)) {
  comp_label <- comparisons$comparison_label[i]
  cond_a <- make.names(comparisons$condition_A[i]); cond_b <- make.names(comparisons$condition_B[i])

  param_a <- paste0("condition", cond_a)
  param_b <- paste0("condition", cond_b)

  if (!(param_a %in% valid_params && param_b %in% valid_params)) {
    message("  - Skipping comparison: ", comp_label, " (one or both conditions excluded or missing)")
    next
  }

  message("  - Testing: ", comp_label, " (", cond_b, " vs ", cond_a, ")")
  contrast_str <- paste0(param_b, " - ", param_a, " = 0")
  
  # Crucial fix: Providing the valid parameter names directly to makeContrast
  L <- makeContrast(contrast_str, parameterNames = valid_params)

  result <- hypothesisTest(object = pe, i = "protein", contrast = L)
  result_df <- as.data.frame(rowData(result[["protein"]])[[colnames(L)]])
  if (adjust_method != "BH") { result_df$adjPval <- p.adjust(result_df$pval, method = adjust_method) }
  result_df$comparison <- comp_label
  result_df$condition_A <- comparisons$condition_A[i]; result_df$condition_B <- comparisons$condition_B[i]
  result_df$protein <- rownames(result_df)
  result_df$significant <- (result_df$adjPval < alpha) & (abs(result_df$logFC) >= lfc_threshold)
  all_results[[comp_label]] <- result_df
}

combined_results <- do.call(rbind, all_results)
if (!is.null(combined_results)) {
  combined_results <- combined_results[order(combined_results$pval), ]
  write.table(combined_results, file = file.path(output_folder, "tmt_results.txt"), sep = "\t", row.names = FALSE, quote = FALSE)
  sig_count <- sum(combined_results$adjPval < alpha, na.rm = TRUE)
  message("\nFound ", sig_count, " significant proteins (adj. p-value < ", alpha, ")")
}

message("\n[QC] Generating quality control plots...")
pdf(file.path(output_folder, "qc_plots.pdf"), width = 14, height = 10)
par(mar=c(12, 4, 4, 2))

peptide_matrix <- assay(pe[[currentAssayName]])
if (ncol(peptide_matrix) > 0 && any(!is.na(peptide_matrix))) {
  boxplot(peptide_matrix, las = 2, main = "Peptide Intensities (Normalized)", ylab = "Log2 Intensity", col = rainbow(ncol(peptide_matrix)), cex.axis = 0.6)
}

protein_matrix <- assay(pe[["protein"]])
if (ncol(protein_matrix) > 0 && any(!is.na(protein_matrix))) {
  boxplot(protein_matrix, las = 2, main = "Protein Intensities (Summarized)", ylab = "Log2 Intensity", col = rainbow(ncol(protein_matrix)), cex.axis = 0.6)
}

pca_data <- t(na.omit(peptide_matrix))
if (nrow(pca_data) > 2 && ncol(pca_data) > 2) {
  col_vars <- apply(pca_data, 2, var, na.rm = TRUE)
  pca_data <- pca_data[, col_vars > 0 & !is.na(col_vars), drop = FALSE]

  if (ncol(pca_data) > 2) {
    pca_result <- prcomp(pca_data, scale. = TRUE)
    pca_variance <- summary(pca_result)$importance[2, 1:2] * 100
    plot(pca_result$x[, 1], pca_result$x[, 2], col = as.numeric(colData(pe)$condition), pch = 19, cex = 2, xlab = paste0("PC1 (", round(pca_variance[1], 1), "%)"), ylab = paste0("PC2 (", round(pca_variance[2], 1), "%)"), main = "PCA - Peptide Level")
    legend("topright", legend = levels(colData(pe)$condition), col = 1:nlevels(colData(pe)$condition), pch = 19, cex = 0.7)
  }
}

if (!is.null(combined_results) && nrow(combined_results) > 0) {
  volcano_data <- combined_results[!is.na(combined_results$logFC) & !is.na(combined_results$adjPval), ]
  if (nrow(volcano_data) > 0) {
    plot(volcano_data$logFC, -log10(volcano_data$adjPval), pch = 20, cex = 0.5, col = ifelse(volcano_data$adjPval < alpha, "red", "gray"), xlab = "Log2 Fold Change", ylab = "-Log10 Adjusted P-value", main = "Volcano Plot")
    abline(h = -log10(alpha), lty = 2, col = "blue")
    if (lfc_threshold > 0) {
      abline(v = c(-lfc_threshold, lfc_threshold), lty = 2, col = "blue")
    }
  }
}
dev.off()

message("\n=== Analysis Complete ===")
message("Results saved to: ", output_folder)
message("  - tmt_results.txt: Differential expression results")
message("  - protein_intensities.txt: Summarized protein intensities")
message("  - normalized_peptides.txt: Normalized peptide intensities")
if (any(grepl("peptideImputed", names(pe)))) message("  - imputed_peptides.txt: Imputed peptide intensities")
if (file.exists(file.path(output_folder, "debug_pre_aggregation.txt"))) message("  - debug_pre_aggregation.txt: Numeric data before aggregation")
message("  - qc_plots.pdf: Quality control plots")
