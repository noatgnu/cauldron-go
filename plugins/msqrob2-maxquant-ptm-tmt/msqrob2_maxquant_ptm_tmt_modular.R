library(QFeatures)
library(msqrob2)
library(limma)
library(MsCoreUtils)
library(SummarizedExperiment)
library(Biostrings)
library(ggplot2)
library(reshape2)

source("src/utils.R")
source("src/ptm_processing.R")
source("src/data_loading.R")
source("src/normalization.R")
source("src/protein_processing.R")
source("src/statistical_analysis.R")
source("src/visualization.R")

args <- commandArgs(trailingOnly = TRUE)
params <- parse_args(args)

input_file <- params$input_file
fasta_file <- params$fasta_file
output_folder <- params$output_folder
annotation_file <- params$annotation_file
annotation_protein_file <- params$annotation_protein_file
comparison_file <- params$comparison_file
feature_id_col <- params$feature_id_col
protein_col <- params$protein_col
site_col <- params$site_col
probability_col <- params$probability_col
min_probability <- ifelse(is.null(params$min_probability), 0.75, as.numeric(params$min_probability))

protein_file <- ifelse(is.null(params$protein_file), NULL, params$protein_file)
protein_id_col <- ifelse(is.null(params$protein_id_col), NULL, params$protein_id_col)
protein_feature_id_col <- ifelse(is.null(params$protein_feature_id_col), NULL, params$protein_feature_id_col)

log2_transform <- !is.null(params$log2_transform) && params$log2_transform != FALSE
analysis_type <- ifelse(is.null(params$analysis_type), "both", params$analysis_type)
filter_min_peptides <- ifelse(is.null(params$filter_min_peptides), 2, as.numeric(params$filter_min_peptides))
filter_min_ptm_sites <- ifelse(is.null(params$filter_min_ptm_sites), 1, as.numeric(params$filter_min_ptm_sites))
col_filter <- ifelse(is.null(params$col_filter), 0.7, as.numeric(params$col_filter))
row_filter <- ifelse(is.null(params$row_filter), 0.7, as.numeric(params$row_filter))
impute_order <- ifelse(is.null(params$impute_order), "after", params$impute_order)
impute_method <- params$impute
normalize_method <- ifelse(is.null(params$normalize_method), "center.median", params$normalize_method)
aggregation_order <- ifelse(is.null(params$aggregation_order), "after", params$aggregation_order)
summarization_method <- ifelse(is.null(params$summarization_method), "robust", params$summarization_method)
remove_shared_peptides <- !is.null(params$remove_shared_peptides) && params$remove_shared_peptides != FALSE
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

message("=== MSqRob2 PTM TMT Analysis ===")

result <- load_and_prepare_ptm_data(
  input_file = input_file,
  annotation_file = annotation_file,
  fasta_file = fasta_file,
  feature_id_col = feature_id_col,
  protein_col = protein_col,
  site_col = site_col,
  probability_col = probability_col,
  min_probability = min_probability,
  col_filter = col_filter,
  row_filter = row_filter,
  exclude_conditions = exclude_conditions
)
pe <- result$pe
protein_col <- result$protein_col

result <- process_ptm_peptidoforms(
  pe = pe,
  impute_order = impute_order,
  impute_method = impute_method,
  normalize_method = normalize_method,
  output_folder = output_folder
)
pe <- result$pe
peptidoformAssayName <- result$peptidoformAssayName

result <- process_global_protein_data(
  pe = pe,
  protein_file = protein_file,
  protein_id_col = protein_id_col,
  protein_feature_id_col = protein_feature_id_col,
  annotation_protein_file = annotation_protein_file,
  protein_col = protein_col,
  log2_transform = log2_transform,
  aggregation_order = aggregation_order,
  impute_order = impute_order,
  impute_method = impute_method,
  normalize_method = normalize_method,
  summarization_method = summarization_method,
  filter_min_peptides = filter_min_peptides,
  col_filter = col_filter,
  row_filter = row_filter,
  remove_shared_peptides = remove_shared_peptides,
  output_folder = output_folder
)
pe <- result$pe
proteinAssayName <- result$proteinAssayName

pe <- perform_model_fitting(
  pe = pe,
  peptidoformAssayName = peptidoformAssayName,
  proteinAssayName = proteinAssayName,
  protein_col = protein_col,
  analysis_type = analysis_type,
  model_run_effect = model_run_effect,
  ridge_penalty = ridge_penalty,
  robust_regression = robust_regression,
  max_iterations = max_iterations
)

results <- perform_hypothesis_testing(
  pe = pe,
  peptidoformAssayName = peptidoformAssayName,
  proteinAssayName = proteinAssayName,
  comparison_file = comparison_file,
  analysis_type = analysis_type,
  alpha = alpha,
  lfc_threshold = lfc_threshold,
  adjust_method = adjust_method,
  output_folder = output_folder
)

generate_qc_plots(
  pe = pe,
  peptidoformAssayName = peptidoformAssayName,
  proteinAssayName = proteinAssayName,
  output_folder = output_folder
)

generate_volcano_plots(
  all_dpa = results$all_dpa,
  all_dpu = results$all_dpu,
  all_protein = results$all_protein,
  alpha = alpha,
  lfc_threshold = lfc_threshold,
  output_folder = output_folder
)

message("\n=== Analysis Complete ===")
message("Results saved to: ", output_folder)
if (file.exists(file.path(output_folder, "dpa_results.txt"))) {
  message("  - dpa_results.txt: Differential PTM abundance (DPA) results")
}
if (file.exists(file.path(output_folder, "dpu_results.txt"))) {
  message("  - dpu_results.txt: Differential PTM usage (DPU) results")
}
if (file.exists(file.path(output_folder, "protein_results.txt"))) {
  message("  - protein_results.txt: Protein-level differential abundance results")
}
message("  - peptidoform_intensities.txt: Normalized peptidoform/PTM intensities")
message("  - protein_intensities.txt: Summarized protein-level intensities")
