library(QFeatures)
library(msqrob2)
library(limma)
library(MsCoreUtils)
library(SummarizedExperiment)
library(Biostrings)
library(ggplot2)
library(reshape2)
library(PhosR)

source("src/utils.R")
source("src/ptm_processing.R")
source("src/data_loading.R")
source("src/normalization.R")
source("src/statistical_analysis.R")
source("src/kinase_analysis.R")
source("src/pathway_analysis.R")
source("src/signaling_analysis.R")
source("src/visualization.R")

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

process_ptm_with_fasta <- function(data, fasta_path, seq_col, protein_col, prob_col, threshold) {
  message("Processing PTM sites using FASTA database...")
  fasta <- readAAStringSet(fasta_path)
  # Clean names: "sp|P12345|NAME" -> "P12345" (Uniprot style)
  names(fasta) <- sapply(strsplit(names(fasta), "|", fixed = TRUE), function(x) if(length(x)>1) x[2] else x[1])

  new_sites <- character(nrow(data))

  for (i in 1:nrow(data)) {
    prot_ids <- strsplit(as.character(data[[protein_col]][i]), ";")[[1]]
    prot_id <- prot_ids[1] # Use leading protein

    seq <- data[[seq_col]][i]
    prob_str <- as.character(data[[prob_col]][i])

    if (is.na(prot_id) || !prot_id %in% names(fasta)) {
      new_sites[i] <- NA
      next
    }

    prot_seq <- as.character(fasta[[prot_id]])
    start_pos <- regexpr(seq, prot_seq, fixed = TRUE)[1]

    if (start_pos == -1) {
      new_sites[i] <- NA
      next
    }

    found_sites <- c()

    if (is.na(prob_str) || prob_str == "") {
        new_sites[i] <- "Unmodified"
        next
    }

    chars <- strsplit(prob_str, "")[[1]]
    curr_seq_idx <- 0
    j <- 1
    while (j <= length(chars)) {
      char <- chars[j]
      if (char == "(") {
        close_paren <- which(chars[j:length(chars)] == ")")[1]
        if (!is.na(close_paren)) {
          prob_val <- as.numeric(paste(chars[(j+1):(j+close_paren-2)], collapse=""))
          if (!is.na(prob_val) && prob_val >= threshold) {
             abs_pos <- start_pos + curr_seq_idx - 1
             residue <- substring(prot_seq, abs_pos, abs_pos)
             found_sites <- c(found_sites, paste0(residue, abs_pos))
          }
          j <- j + close_paren
        } else {
          j <- j + 1
        }
      } else {
        curr_seq_idx <- curr_seq_idx + 1
        j <- j + 1
      }
    }

    if (length(found_sites) > 0) {
      # Format: Protein_Site1_Site2
      new_sites[i] <- paste(c(prot_id, found_sites), collapse="_")
    } else {
      new_sites[i] <- "Unmodified"
    }
  }
  return(new_sites)
}

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

organism <- ifelse(is.null(params$organism), "human", params$organism)
kinase_num_motifs <- ifelse(is.null(params$kinase_num_motifs), 5, as.numeric(params$kinase_num_motifs))
top_diff_sites_heatmap <- ifelse(is.null(params$top_diff_sites_heatmap), 50, as.numeric(params$top_diff_sites_heatmap))
top_phosphosite_heatmap <- ifelse(is.null(params$top_phosphosite_heatmap), 50, as.numeric(params$top_phosphosite_heatmap))

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

message("\n[1/8] Reading data...")
input_sep <- detect_delimiter(input_file)
peptide_data <- read.table(input_file, sep = input_sep, header = TRUE,
                           na.strings = c("NA", "NaN", "N/A", "#VALUE!", ""),
                           check.names = FALSE, stringsAsFactors = FALSE)

message("[2/8] Reading annotation...")
annot_sep <- detect_delimiter(annotation_file)
annotation <- read.table(annotation_file, sep = annot_sep, header = TRUE,
                        stringsAsFactors = FALSE, check.names = FALSE)

colnames(peptide_data) <- make.names(colnames(peptide_data))
annotation$Sample <- make.names(annotation$Sample)
feature_id_col <- make.names(feature_id_col)
protein_col <- make.names(protein_col)
site_col <- make.names(site_col)
if (!is.null(probability_col)) probability_col <- make.names(probability_col)

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

# Construct Feature ID
if (!is.null(fasta_file) && file.exists(fasta_file) && !is.null(probability_col)) {
  parsed_sites <- process_ptm_with_fasta(peptide_data, fasta_file, feature_id_col, protein_col, probability_col, min_probability)
  peptide_data$parsed_site <- parsed_sites
  
  # Filter out records not found in FASTA
  peptide_data <- peptide_data[!is.na(peptide_data$parsed_site), ]
  message("  - Records remaining after FASTA filtering: ", nrow(peptide_data))
  
  peptide_data$clean_feature_id <- paste(peptide_data$parsed_site, peptide_data[[feature_id_col]], peptide_data[[protein_col]], sep = "_")
} else {
  if (!site_col %in% colnames(peptide_data)) stop("Site column not found")
  peptide_data$clean_feature_id <- paste(peptide_data[[site_col]], peptide_data[[feature_id_col]], peptide_data[[protein_col]], sep = "_")
}

rownames(peptide_data) <- make.unique(peptide_data$clean_feature_id)

message("[3/8] Creating QFeatures object...")
quant_cols <- which(colnames(peptide_data) %in% all_good_sample_cols)

pe <- readQFeatures(
  peptide_data,
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
if ("Run" %in% colnames(annotation)) {
  colData_df_all$run <- factor(make.names(annotation$Run[match(all_good_sample_cols, annotation$Sample)]))
}

colData(pe) <- DataFrame(colData_df_all)
pe <- pe[, rownames(colData_df_all)]

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

message("[4/8] Filtering and aggregating PSMs to Peptidoforms...")
pe <- zeroIsNA(pe, i = "peptideRaw")
pe <- filterNA(pe, i = "peptideRaw", pNA = row_filter)

# Filter by Protein presence
keep_protein <- !is.na(rowData(pe[["peptideRaw"]])[[protein_col]])
pe <- pe[keep_protein, , ]

# Aggregation 1: PSM -> Peptidoform (PTM Feature)
message("  - Aggregating PSMs to Peptidoforms...")
pe <- aggregateFeatures(pe, i = "peptideRaw", fcol = "clean_feature_id", name = "peptidoform", fun = MsCoreUtils::robustSummary)

# ============================================================================
# Helper Functions
# ============================================================================

do_imputation <- function(assayName) {
  if (!is.null(impute_method) && impute_method != "none") {
    pe <<- impute(pe, method = impute_method, i = assayName, name = paste0(assayName, "Imputed"))
    return(paste0(assayName, "Imputed"))
  }
  return(assayName)
}

do_normalization <- function(assayName) {
  if (normalize_method != "none") {
    pe <<- normalize(pe, i = assayName, name = paste0(assayName, "Norm"), method = normalize_method)
    return(paste0(assayName, "Norm"))
  } else {
    pe <<- addAssay(pe, pe[[assayName]], name = paste0(assayName, "Norm"))
    return(paste0(assayName, "Norm"))
  }
}

do_aggregation <- function(assayName) {
  message("  - Aggregating Peptidoforms to Proteins (method: ", summarization_method, ")...")

  if (!protein_col %in% colnames(rowData(pe[[assayName]]))) {
    available_cols <- colnames(rowData(pe[[assayName]]))
    stop("Protein column '", protein_col, "' not found in rowData. Available columns: ", paste(available_cols, collapse=", "))
  }

  protein_values <- rowData(pe[[assayName]])[[protein_col]]

  # Check if protein column has numeric suffixes (like P00918.42)
  if (any(grepl("\\.[0-9]+$", protein_values))) {
    message("  - WARNING: Protein column contains numeric suffixes! Cleaning...")
    # Remove the numeric suffixes
    clean_proteins <- sub("\\.[0-9]+$", "", protein_values)
    rowData(pe[[assayName]])[[protein_col]] <- clean_proteins
    message("  - Cleaned protein values")
  }

  # Filter proteins by minimum peptide count (same as non-PTM workflow)
  peptide_counts <- table(rowData(pe[[assayName]])[[protein_col]])
  proteins_to_keep <- names(peptide_counts)[peptide_counts >= filter_min_peptides]
  keep_min_pep <- rowData(pe[[assayName]])[[protein_col]] %in% proteins_to_keep
  pe <<- pe[keep_min_pep, , ]

  if (nrow(pe[[assayName]]) == 0) {
    stop("No features remaining after protein-level filtering (minimum ", filter_min_peptides, " peptides per protein)")
  }

  message("  - ", nrow(pe[[assayName]]), " peptidoforms for ", length(proteins_to_keep), " proteins (min ", filter_min_peptides, " peptides/protein)")

  summ_fun <- switch(summarization_method,
    "robust" = MsCoreUtils::robustSummary,
    "sum" = colSums,
    "mean" = colMeans,
    "median" = matrixStats::colMedians,
    MsCoreUtils::robustSummary
  )

  # Subset QFeatures to current assay columns (important for proper aggregation)
  pe <<- pe[, colnames(pe[[assayName]])]

  # Aggregate features by protein
  pe <<- aggregateFeatures(pe, i = assayName, fcol = protein_col, name = "protein", fun = summ_fun)

  message("  - Result: ", nrow(pe[["protein"]]), " proteins")

  prot_rownames <- rownames(pe[["protein"]])
  if (any(grepl("\\.[0-9]+$", prot_rownames))) {
    message("  - WARNING: Protein rownames still have numeric suffixes after aggregation!")
  }

  return("protein")
}

# ============================================================================
# [5/8] PTM Data Processing
# ============================================================================
message("[5/8] Processing PTM peptidoform data...")

# Process peptidoforms: normalize, impute, log transform
currentAssayName <- "peptidoform"

# Save raw for plot
pe <- addAssay(pe, pe[[currentAssayName]], name = "peptidoform_raw")

if (impute_order == "before") {
  message("  - Imputation (method: ", ifelse(is.null(impute_method) || impute_method == "none", "none", impute_method), ")...")
  peptidoformAssayName <- do_imputation(currentAssayName)
  message("  - Normalization (method: ", normalize_method, ")...")
  peptidoformAssayName <- do_normalization(peptidoformAssayName)
} else {
  message("  - Normalization (method: ", normalize_method, ")...")
  peptidoformAssayName <- do_normalization(currentAssayName)
  message("  - Imputation (method: ", ifelse(is.null(impute_method) || impute_method == "none", "none", impute_method), ")...")
  peptidoformAssayName <- do_imputation(peptidoformAssayName)
}

message("  - Log2 transformation...")
pe <- logTransform(pe, base = 2, i = peptidoformAssayName, name = "peptidoformLog")
peptidoformAssayName <- "peptidoformLog"

message("  - Saving peptidoform intensities...")
normalized_peptides_out <- cbind(as.data.frame(rowData(pe[[peptidoformAssayName]])), as.data.frame(assay(pe[[peptidoformAssayName]])))
list_cols <- sapply(normalized_peptides_out, is.list)
if (any(list_cols)) normalized_peptides_out <- normalized_peptides_out[, !list_cols, drop = FALSE]
write.table(normalized_peptides_out, file = file.path(output_folder, "peptidoform_intensities.txt"), sep = "\t", row.names = FALSE, quote = FALSE)

message("  - PTM peptidoform processing complete: ", nrow(pe[[peptidoformAssayName]]), " features")

# ============================================================================
# [6/8] Global Protein Data Processing
# ============================================================================
message("[6/8] Processing global protein data...")

use_external_protein <- !is.null(protein_file) && file.exists(protein_file)

# Refined helper functions matching non-PTM logic
process_impute <- function(pe_obj, i, name, method) {
  if (!is.null(method) && method != "none") {
    pe_obj <- impute(pe_obj, method = method, i = i, name = name)
  } else {
    pe_obj <- addAssay(pe_obj, pe_obj[[i]], name = name)
  }
  return(pe_obj)
}

process_norm <- function(pe_obj, i, name, method) {
  if (method != "none") {
    pe_obj <- normalize(pe_obj, i = i, name = name, method = method)
  } else {
    pe_obj <- addAssay(pe_obj, pe_obj[[i]], name = name)
  }
  return(pe_obj)
}

process_agg <- function(pe_obj, i, fcol, name, method, min_pep) {
  peptide_counts <- table(rowData(pe_obj[[i]])[[fcol]])
  proteins_to_keep <- names(peptide_counts)[peptide_counts >= min_pep]
  pe_obj <- pe_obj[rowData(pe_obj[[i]])[[fcol]] %in% proteins_to_keep, , ]
  
  if (nrow(pe_obj[[i]]) == 0) stop("No features remaining after filtering in global protein data")

  summ_fun <- switch(method, 
    "robust" = MsCoreUtils::robustSummary, 
    "sum" = colSums, "mean" = colMeans, "median" = matrixStats::colMedians, 
    MsCoreUtils::robustSummary
  )
  
  pe_obj <- pe_obj[, colnames(pe_obj[[i]])]
  pe_obj <- aggregateFeatures(pe_obj, i = i, fcol = fcol, name = name, fun = summ_fun)
  return(pe_obj)
}

if (use_external_protein) {
  message("  - Loading external protein file...")
  protein_sep <- detect_delimiter(protein_file)
  protein_data <- read.table(protein_file, sep = protein_sep, header = TRUE, 
                             na.strings = c("NA", "NaN", "N/A", "#VALUE!", ""), 
                             check.names = FALSE, stringsAsFactors = FALSE)
  
  colnames(protein_data) <- make.names(colnames(protein_data))
  safe_prot_id <- make.names(protein_id_col)
  safe_prot_feature_id <- make.names(protein_feature_id_col)

  if (!is.null(annotation_protein_file) && file.exists(annotation_protein_file)) {
     message("  - Loading protein annotation file...")
     annot_prot_sep <- detect_delimiter(annotation_protein_file)
     annotation_prot <- read.table(annotation_protein_file, sep = annot_prot_sep, header = TRUE, 
                                   stringsAsFactors = FALSE, check.names = FALSE)
     annotation_prot$Sample <- make.names(annotation_prot$Sample)
     annotation_prot$Condition <- make.names(annotation_prot$Condition)
     if ("BioReplicate" %in% colnames(annotation_prot)) annotation_prot$BioReplicate <- make.names(annotation_prot$BioReplicate)
  } else {
     message("  - No protein annotation provided, assuming same sample names as PTM...")
     annotation_prot <- annotation
  }

  ptm_annot <- as.data.frame(colData(pe))
  ptm_annot$sample <- rownames(ptm_annot)
  if (!"biorep" %in% colnames(ptm_annot)) ptm_annot$biorep <- ptm_annot$sample
  if (!"BioReplicate" %in% colnames(annotation_prot)) annotation_prot$BioReplicate <- annotation_prot$Sample

  ptm_keys <- paste(ptm_annot$condition, ptm_annot$biorep, sep = "_")
  prot_keys <- paste(annotation_prot$Condition, annotation_prot$BioReplicate, sep = "_")
  
  common_keys <- intersect(ptm_keys, prot_keys)
  if (length(common_keys) == 0) stop("No matching samples between PTM and Protein annotation")

  final_ptm_samples <- ptm_annot$sample[match(common_keys, ptm_keys)]
  final_prot_samples <- annotation_prot$Sample[match(common_keys, prot_keys)]
  
  valid_indices <- final_prot_samples %in% colnames(protein_data)
  final_ptm_samples <- final_ptm_samples[valid_indices]
  final_prot_samples <- final_prot_samples[valid_indices]

  message("  - Matched ", length(final_ptm_samples), " samples between experiments")

  protein_data_subset <- protein_data[, c(safe_prot_id, safe_prot_feature_id, final_prot_samples)]
  colnames(protein_data_subset)[match(final_prot_samples, colnames(protein_data_subset))] <- final_ptm_samples

  protein_unique_ids <- paste(protein_data_subset[[safe_prot_feature_id]], protein_data_subset[[safe_prot_id]], sep = "_")
  rownames(protein_data_subset) <- make.unique(as.character(protein_unique_ids))

  pe_prot <- readQFeatures(protein_data_subset, quantCols = final_ptm_samples, name = "peptideRaw")
  colData(pe_prot) <- colData(pe[, final_ptm_samples])
  
  pe_prot <- zeroIsNA(pe_prot, i = "peptideRaw")
  
  na_prop <- colMeans(is.na(assay(pe_prot[["peptideRaw"]])))
  keep_samples_prot <- na_prop < col_filter
  if (sum(!keep_samples_prot) > 0) {
    message("  - Removing ", sum(!keep_samples_prot), " samples with high NA")
    pe_prot <- pe_prot[, keep_samples_prot]
  }
  pe_prot <- filterNA(pe_prot, i = "peptideRaw", pNA = row_filter)
  
  if (remove_shared_peptides) {
    shared_prot <- grepl(";", rowData(pe_prot[["peptideRaw"]])[[safe_prot_id]]) | grepl(",", rowData(pe_prot[["peptideRaw"]])[[safe_prot_id]])
    pe_prot <- pe_prot[!shared_prot, , ]
  }

  current_prot <- "peptideRaw"
  if (log2_transform) {
    pe_prot <- logTransform(pe_prot, base = 2, i = current_prot, name = "peptideLog")
    current_prot <- "peptideLog"
  }

  if (aggregation_order == "before") {
    if (impute_order == "before") pe_prot <- process_impute(pe_prot, current_prot, "psmImp", impute_method)
    pe_prot <- process_agg(pe_prot, current_prot, safe_prot_id, "protein", summarization_method, filter_min_peptides)
    current_prot <- "protein"
    
    # Save raw for plot
    pe <- addAssay(pe, pe_prot[[current_prot]], name = "protein_raw")
    
    pe_prot <- process_norm(pe_prot, current_prot, "proteinNorm", normalize_method)
    current_prot <- "proteinNorm"
    if (impute_order == "after") pe_prot <- process_impute(pe_prot, current_prot, "proteinImp", impute_method)
  } else {
    if (impute_order == "before") pe_prot <- process_impute(pe_prot, current_prot, "psmImp", impute_method)
    pe_prot <- process_norm(pe_prot, current_prot, "psmNorm", normalize_method)
    current_prot <- "psmNorm"
    if (impute_order == "after") pe_prot <- process_impute(pe_prot, current_prot, "psmImpAfter", impute_method)
    current_prot <- "psmImpAfter"
    pe_prot <- process_agg(pe_prot, current_prot, safe_prot_id, "protein", summarization_method, filter_min_peptides)
    current_prot <- "protein"
    
    # Save raw for plot
    pe <- addAssay(pe, pe_prot[[current_prot]], name = "protein_raw")
  }

  pe <- pe[, colnames(pe_prot)]
  pe <- addAssay(pe, pe_prot[["protein"]], name = "protein_processed")
  proteinAssayName <- "protein_processed"

} else {
  message("  - Aggregating global protein abundance from PTM PSMs...")
  pe_prot <- QFeatures(list(peptideRaw = pe[["peptideRaw"]]))
  colData(pe_prot) <- colData(pe)
  
  current_prot <- "peptideRaw"
  if (log2_transform) {
    pe_prot <- logTransform(pe_prot, base = 2, i = current_prot, name = "peptideLog")
    current_prot <- "peptideLog"
  }

  if (aggregation_order == "before") {
    if (impute_order == "before") pe_prot <- process_impute(pe_prot, current_prot, "psmImp", impute_method)
    pe_prot <- process_agg(pe_prot, current_prot, protein_col, "protein", summarization_method, filter_min_peptides)
    current_prot <- "protein"
    
    # Save raw for plot
    pe <- addAssay(pe, pe_prot[[current_prot]], name = "protein_raw")
    
    pe_prot <- process_norm(pe_prot, current_prot, "proteinNorm", normalize_method)
    current_prot <- "proteinNorm"
    if (impute_order == "after") pe_prot <- process_impute(pe_prot, current_prot, "proteinImp", impute_method)
  } else {
    if (impute_order == "before") pe_prot <- process_impute(pe_prot, current_prot, "psmImp", impute_method)
    pe_prot <- process_norm(pe_prot, current_prot, "psmNorm", normalize_method)
    current_prot <- "psmNorm"
    if (impute_order == "after") pe_prot <- process_impute(pe_prot, current_prot, "psmImpAfter", impute_method)
    current_prot <- "psmImpAfter"
    pe_prot <- process_agg(pe_prot, current_prot, protein_col, "protein", summarization_method, filter_min_peptides)
    current_prot <- "protein"
    
    # Save raw for plot
    pe <- addAssay(pe, pe_prot[[current_prot]], name = "protein_raw")
  }

  pe <- addAssay(pe, pe_prot[["protein"]], name = "protein_processed")
  proteinAssayName <- "protein_processed"
}

message("  - Saving global protein intensities...")
protein_intensities <- cbind(
  as.data.frame(rowData(pe[[proteinAssayName]])),
  as.data.frame(assay(pe[[proteinAssayName]]))
)
list_cols <- sapply(protein_intensities, is.list)
if (any(list_cols)) { protein_intensities <- protein_intensities[, !list_cols, drop = FALSE] }
write.table(protein_intensities, file = file.path(output_folder, "protein_intensities.txt"), sep = "\t", row.names = FALSE, quote = FALSE)

if (use_external_protein) {
  message("  - Global protein processing complete (from separate experiment)")
} else {
  message("  - Global protein processing complete (from PTM experiment)")
}

# ============================================================================
# [7/8] Statistical Model Fitting
# ============================================================================
message("[7/8] Fitting statistical models...")
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

tryCatch({
  pe <- msqrob(object = pe, i = proteinAssayName, formula = formula, ridge = ridge_penalty, robust = robust_regression, maxitRob = max_iterations)
}, error = function(e) {
  stop("Protein model fitting failed with error: ", e$message)
})

if (!"msqrobModels" %in% colnames(rowData(pe[[proteinAssayName]]))) {
  stop("Protein model fitting failed. Check if there are enough observations per condition.")
}

protein_models <- rowData(pe[[proteinAssayName]])$msqrobModels
n_null_models <- sum(sapply(protein_models, is.null))
if (n_null_models > 0) {
  message("  - Warning: ", n_null_models, " proteins failed to fit. They will be excluded from testing.")
}

tryCatch({
  pe <- msqrob(object = pe, i = peptidoformAssayName, formula = formula, ridge = ridge_penalty, robust = robust_regression, maxitRob = max_iterations)
}, error = function(e) {
  stop("PTM model fitting failed with error: ", e$message)
})

if (!"msqrobModels" %in% colnames(rowData(pe[[peptidoformAssayName]]))) {
  stop("PTM model fitting failed. Check if there are enough observations per condition.")
}

ptm_models <- rowData(pe[[peptidoformAssayName]])$msqrobModels
n_null_models_ptm <- sum(sapply(ptm_models, is.null))
if (n_null_models_ptm > 0) {
  message("  - Warning: ", n_null_models_ptm, " PTM features failed to fit. They will be excluded from testing.")
}

if (analysis_type %in% c("DPU", "both")) {
  ptm_prot_ids <- rowData(pe[[peptidoformAssayName]])[[protein_col]]

  ptm_mat <- assay(pe[[peptidoformAssayName]])
  prot_mat <- assay(pe[[proteinAssayName]])

  # Match proteins
  m <- match(ptm_prot_ids, rownames(pe[[proteinAssayName]]))
  valid_dpu <- !is.na(m)

  if (sum(valid_dpu) > 0) {
    dpu_mat <- ptm_mat[valid_dpu, ] - prot_mat[m[valid_dpu], ]

    # Create SE with clean rowData (exclude msqrobModels column from peptidoform)
    ptm_rowdata <- rowData(pe[[peptidoformAssayName]])[valid_dpu, ]
    if ("msqrobModels" %in% colnames(ptm_rowdata)) {
      ptm_rowdata <- ptm_rowdata[, colnames(ptm_rowdata) != "msqrobModels", drop = FALSE]
    }
    dpu_se <- SummarizedExperiment(assays=list(dpu=dpu_mat), rowData=ptm_rowdata)
    colData(dpu_se) <- colData(pe)
    pe <- addAssay(pe, dpu_se, name = "peptideDPU")

    pe <- msqrob(object = pe, i = "peptideDPU", formula = formula, ridge = ridge_penalty, robust = robust_regression, maxitRob = max_iterations)
  }
}

message("[7/8] Hypothesis testing...")

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

all_dpa <- list()
all_dpu <- list()
all_protein <- list()
valid_params <- paste0("condition", levels(colData(pe)$condition))

# Filter out features with NULL models before hypothesis testing
if (analysis_type %in% c("DPA", "both")) {
  ptm_models <- rowData(pe[[peptidoformAssayName]])$msqrobModels
  valid_ptm <- !sapply(ptm_models, is.null)
  if (sum(valid_ptm) == 0) {
    stop("No valid PTM models for hypothesis testing. All features failed to fit.")
  }
  if (sum(!valid_ptm) > 0) {
    message("  - Filtering ", sum(!valid_ptm), " PTM features with failed models")
    # Filter the peptidoform assay only
    ptm_assay_filtered <- pe[[peptidoformAssayName]][valid_ptm, ]
    # Replace peptidoform assay with filtered version
    pe <- removeAssay(pe, peptidoformAssayName)
    pe <- addAssay(pe, ptm_assay_filtered, name = peptidoformAssayName)
  }
}

protein_models_valid <- rowData(pe[[proteinAssayName]])$msqrobModels
valid_proteins <- !sapply(protein_models_valid, is.null)
if (sum(valid_proteins) == 0) {
  stop("No valid protein models for hypothesis testing. All features failed to fit.")
}
if (sum(!valid_proteins) > 0) {
  message("  - Filtering ", sum(!valid_proteins), " proteins with failed models")
  # Filter the protein assay only
  protein_assay_filtered <- pe[[proteinAssayName]][valid_proteins, ]
  # Create new QFeatures with filtered protein assay
  pe_protein_filtered <- pe
  pe_protein_filtered <- removeAssay(pe_protein_filtered, proteinAssayName)
  pe_protein_filtered <- addAssay(pe_protein_filtered, protein_assay_filtered, name = proteinAssayName)
} else {
  pe_protein_filtered <- pe
}

for (i in 1:nrow(comparisons)) {
  comp_label <- comparisons$comparison_label[i]
  cond_a <- make.names(comparisons$condition_A[i])
  cond_b <- make.names(comparisons$condition_B[i])

  param_a <- paste0("condition", cond_a)
  param_b <- paste0("condition", cond_b)

  if (!(param_a %in% valid_params && param_b %in% valid_params)) {
    message("  - Skipping comparison: ", comp_label, " (one or both conditions excluded or missing)")
    next
  }

  message("  - Testing: ", comp_label, " (", cond_b, " vs ", cond_a, ")")
  contrast_str <- paste0(param_b, " - ", param_a, " = 0")
  L <- makeContrast(contrast_str, parameterNames = valid_params)

  # DPA (Differential Peptide/PTM Abundance)
  if (analysis_type %in% c("DPA", "both")) {
    result <- hypothesisTest(object = pe, i = peptidoformAssayName, contrast = L)
    result_df <- as.data.frame(rowData(result[[peptidoformAssayName]])[[colnames(L)]])
    if (adjust_method != "BH") { result_df$adjPval <- p.adjust(result_df$pval, method = adjust_method) }
    result_df$comparison <- comp_label
    result_df$condition_A <- comparisons$condition_A[i]
    result_df$condition_B <- comparisons$condition_B[i]
    result_df$feature <- rownames(result_df)
    result_df$significant <- (result_df$adjPval < alpha) & (abs(result_df$logFC) >= lfc_threshold)
    all_dpa[[comp_label]] <- result_df
  }

  # DPU (Differential Peptide/PTM Usage)
  if (analysis_type %in% c("DPU", "both") && "peptideDPU" %in% names(pe)) {
    result <- hypothesisTest(object = pe, i = "peptideDPU", contrast = L)
    result_df <- as.data.frame(rowData(result[["peptideDPU"]])[[colnames(L)]])
    if (adjust_method != "BH") { result_df$adjPval <- p.adjust(result_df$pval, method = adjust_method) }
    result_df$comparison <- comp_label
    result_df$condition_A <- comparisons$condition_A[i]
    result_df$condition_B <- comparisons$condition_B[i]
    result_df$feature <- rownames(result_df)
    result_df$significant <- (result_df$adjPval < alpha) & (abs(result_df$logFC) >= lfc_threshold)
    all_dpu[[comp_label]] <- result_df
  }

  # Protein-level results
  result <- hypothesisTest(object = pe_protein_filtered, i = proteinAssayName, contrast = L)
  result_df <- as.data.frame(rowData(result[[proteinAssayName]])[[colnames(L)]])
  if (adjust_method != "BH") { result_df$adjPval <- p.adjust(result_df$pval, method = adjust_method) }
  result_df$comparison <- comp_label
  result_df$condition_A <- comparisons$condition_A[i]
  result_df$condition_B <- comparisons$condition_B[i]
  result_df$protein <- rownames(result_df)
  result_df$significant <- (result_df$adjPval < alpha) & (abs(result_df$logFC) >= lfc_threshold)
  all_protein[[comp_label]] <- result_df
}

if (length(all_dpa) > 0) {
  dpa_final <- do.call(rbind, all_dpa)
  dpa_final <- dpa_final[order(dpa_final$pval), ]
  write.table(dpa_final, file.path(output_folder, "dpa_results.txt"), sep="\t", quote=FALSE, row.names=FALSE)
  sig_count <- sum(dpa_final$adjPval < alpha, na.rm = TRUE)
  message("  - DPA: Found ", sig_count, " significant PTM features (adj. p-value < ", alpha, ")")
}

if (length(all_dpu) > 0) {
  dpu_final <- do.call(rbind, all_dpu)
  dpu_final <- dpu_final[order(dpu_final$pval), ]
  write.table(dpu_final, file.path(output_folder, "dpu_results.txt"), sep="\t", quote=FALSE, row.names=FALSE)
  sig_count <- sum(dpu_final$adjPval < alpha, na.rm = TRUE)
  message("  - DPU: Found ", sig_count, " significant PTM features (adj. p-value < ", alpha, ")")
}

if (length(all_protein) > 0) {
  protein_final <- do.call(rbind, all_protein)
  protein_final <- protein_final[order(protein_final$pval), ]
  write.table(protein_final, file.path(output_folder, "protein_results.txt"), sep="\t", quote=FALSE, row.names=FALSE)
  sig_count <- sum(protein_final$adjPval < alpha, na.rm = TRUE)
  message("  - Protein: Found ", sig_count, " significant proteins (adj. p-value < ", alpha, ")")
}

message("[8/8] Generating quality control and volcano plots...")

# Helper for ggplot boxplots
plot_boxplot_ggplot <- function(pe_obj, i, title) {
  mat <- assay(pe_obj[[i]])
  if (ncol(mat) == 0) return(NULL)
  df_long <- reshape2::melt(mat)
  colnames(df_long) <- c("Feature", "Sample", "Intensity")
  cd <- as.data.frame(colData(pe_obj))
  cd$Sample <- rownames(cd)
  df_long <- merge(df_long, cd, by = "Sample")
  p <- ggplot(df_long, aes(x = Sample, y = Intensity, fill = condition)) +
    geom_boxplot(outlier.size = 0.5) + theme_minimal() +
    theme(axis.text.x = element_text(angle = 90, vjust = 0.5, hjust=1)) +
    labs(title = title, y = "Log2 Intensity")
  return(p)
}

# Helper for volcano plots
plot_volcano <- function(df, title, comparison) {
  if (is.null(df) || nrow(df) == 0) return(NULL)
  v_data <- df[!is.na(df$logFC) & !is.na(df$adjPval), ]
  if (nrow(v_data) == 0) return(NULL)
  p <- ggplot(v_data, aes(x = logFC, y = -log10(adjPval), color = significant)) +
    geom_point(alpha = 0.5, size = 0.8) +
    scale_color_manual(values = c("gray", "red")) +
    theme_minimal() +
    geom_hline(yintercept = -log10(alpha), linetype = "dashed", color = "blue") +
    {if(lfc_threshold > 0) geom_vline(xintercept = c(-lfc_threshold, lfc_threshold), linetype = "dashed", color = "blue")} +
    labs(title = paste(title, ":", comparison))
  return(p)
}

# --- 1. GLOBAL PROTEIN QC PLOTS ---
pdf(file.path(output_folder, "protein_qc_plots.pdf"), width = 14, height = 10)
if ("protein_raw" %in% names(pe)) {
  p <- plot_boxplot_ggplot(pe, "protein_raw", "Protein Intensities (Before Norm)")
  if (!is.null(p)) print(p)
}
p <- plot_boxplot_ggplot(pe, proteinAssayName, "Protein Intensities (Global Normalized)")
if (!is.null(p)) print(p)

# Protein PCA
protein_matrix <- assay(pe[[proteinAssayName]])
pca_data_prot <- t(na.omit(protein_matrix))
if (nrow(pca_data_prot) > 2 && ncol(pca_data_prot) > 2) {
  col_vars_prot <- apply(pca_data_prot, 2, var, na.rm = TRUE)
  pca_data_prot <- pca_data_prot[, col_vars_prot > 0 & !is.na(col_vars_prot), drop = FALSE]
  if (ncol(pca_data_prot) > 2) {
    pca_result_prot <- prcomp(pca_data_prot, scale. = TRUE)
    df_pca_prot <- as.data.frame(pca_result_prot$x)
    df_pca_prot$Sample <- rownames(df_pca_prot)
    cd <- as.data.frame(colData(pe))
    cd$Sample <- rownames(cd)
    df_pca_prot <- merge(df_pca_prot, cd, by = "Sample")
    p_pca_prot <- ggplot(df_pca_prot, aes(x = PC1, y = PC2, color = condition, label = Sample)) +
      geom_point(size = 3) + geom_text(vjust = 1.5, size = 3) + theme_minimal() +
      labs(title = "PCA - Protein Level")
    print(p_pca_prot)
  }
}
dev.off()

# --- 2. GLOBAL PROTEIN VOLCANO PLOTS ---
pdf(file.path(output_folder, "protein_volcano_plots.pdf"), width = 14, height = 10)
for (comp in names(all_protein)) {
  print(plot_volcano(all_protein[[comp]], "Volcano Plot - Protein Level", comp))
}
dev.off()

# --- 3. PTM QC PLOTS ---
pdf(file.path(output_folder, "ptm_qc_plots.pdf"), width = 14, height = 10)
if ("peptidoform_raw" %in% names(pe)) {
  p <- plot_boxplot_ggplot(pe, "peptidoform_raw", "PTM/Peptidoform Intensities (Before Norm)")
  if (!is.null(p)) print(p)
}
p <- plot_boxplot_ggplot(pe, peptidoformAssayName, "PTM/Peptidoform Intensities (Normalized)")
if (!is.null(p)) print(p)

# Peptidoform PCA
peptidoform_matrix <- assay(pe[[peptidoformAssayName]])
pca_data <- t(na.omit(peptidoform_matrix))
if (nrow(pca_data) > 2 && ncol(pca_data) > 2) {
  col_vars <- apply(pca_data, 2, var, na.rm = TRUE)
  pca_data <- pca_data[, col_vars > 0 & !is.na(col_vars), drop = FALSE]
  if (ncol(pca_data) > 2) {
    pca_result <- prcomp(pca_data, scale. = TRUE)
    df_pca <- as.data.frame(pca_result$x)
    df_pca$Sample <- rownames(df_pca)
    cd <- as.data.frame(colData(pe))
    cd$Sample <- rownames(cd)
    df_pca <- merge(df_pca, cd, by = "Sample")
    p_pca <- ggplot(df_pca, aes(x = PC1, y = PC2, color = condition, label = Sample)) +
      geom_point(size = 3) + geom_text(vjust = 1.5, size = 3) + theme_minimal() +
      labs(title = "PCA - PTM/Peptidoform Level")
    print(p_pca)
  }
}
dev.off()

# --- 4. PTM VOLCANO PLOTS (DPA/DPU) ---
pdf(file.path(output_folder, "ptm_volcano_plots.pdf"), width = 14, height = 10)
comp_names <- names(all_protein)
if (length(comp_names) > 0) {
  for (comp in comp_names) {
    if (exists("all_dpa") && comp %in% names(all_dpa)) print(plot_volcano(all_dpa[[comp]], "Volcano Plot - DPA (PTM Abundance)", comp))
    if (exists("all_dpu") && comp %in% names(all_dpu)) print(plot_volcano(all_dpu[[comp]], "Volcano Plot - DPU (PTM Usage)", comp))
  }
}
dev.off()

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
message("  - qc_plots.pdf: Quality control plots")

message("\n=== PhosR-Specific Analysis ===")
library(PhosR)
library(Biostrings)

source("src/kinase_analysis.R")
source("src/pathway_analysis.R")
source("src/signaling_analysis.R")
source("src/visualization.R")

message("\nExtracting data from MSqRob2 results...")

phosr_assay_name <- NULL
if (analysis_type %in% c("DPU", "both") && "peptideDPU" %in% names(pe)) {
  phosr_assay_name <- "peptideDPU"
  message("  - Using DPU-normalized assay for PhosR analysis")
} else {
  phosr_assay_name <- peptidoformAssayName
  message("  - Using DPA assay for PhosR analysis")
}

ptm_assay <- assay(pe[[phosr_assay_name]])
ptm_rowdata_original <- rowData(pe[[phosr_assay_name]])
ptm_coldata <- colData(pe)

message("  - Extracted from assay '", phosr_assay_name, "': ", nrow(ptm_assay), " sites x ", ncol(ptm_assay), " samples")

fasta_parsed <- NULL
accession_to_gene <- NULL

if (!is.null(fasta_file) && file.exists(fasta_file)) {
  message("  - Reading and parsing FASTA file...")
  fasta_parsed <- readAAStringSet(fasta_file)

  fasta_headers <- names(fasta_parsed)
  accession_to_gene <- list()

  for (header in fasta_headers) {
    parts <- strsplit(header, "\\|", fixed = FALSE)[[1]]
    if (length(parts) >= 3) {
      accession <- parts[2]
      gene_info <- parts[3]
      gene_symbol <- strsplit(gene_info, "_")[[1]][1]
      accession_to_gene[[accession]] <- toupper(gene_symbol)
    } else if (length(parts) == 2) {
      accession <- parts[2]
      accession_to_gene[[accession]] <- toupper(accession)
    }
  }

  names(fasta_parsed) <- sapply(strsplit(fasta_headers, "|", fixed = TRUE), function(x) if(length(x)>1) x[2] else x[1])
  message("  - Extracted ", length(accession_to_gene), " gene symbol mappings from FASTA")
  message("  - Sample mappings (first 3): ")
  for (i in 1:min(3, length(accession_to_gene))) {
    acc <- names(accession_to_gene)[i]
    gene <- accession_to_gene[[acc]]
    message("    ", acc, " -> ", gene)
  }
}

phosr_protein <- character(nrow(ptm_rowdata_original))
phosr_sequence <- character(nrow(ptm_rowdata_original))
phosr_site_id <- character(nrow(ptm_rowdata_original))
phosr_gene <- character(nrow(ptm_rowdata_original))

for (i in seq_len(nrow(ptm_rowdata_original))) {
  protein_raw <- as.character(ptm_rowdata_original[[protein_col]][i])

  if (!is.na(protein_raw) && protein_raw != "") {
    protein_list <- strsplit(protein_raw, ";")[[1]]
    main_protein <- protein_list[1]
    main_protein <- gsub("^sp\\||^tr\\|", "", main_protein)
    main_protein <- sub("\\|.*", "", main_protein)
    phosr_protein[i] <- main_protein

    if (!is.null(accession_to_gene) && !is.null(accession_to_gene[[main_protein]])) {
      phosr_gene[i] <- accession_to_gene[[main_protein]]
    } else {
      phosr_gene[i] <- toupper(main_protein)
    }
  } else {
    phosr_protein[i] <- NA_character_
    phosr_gene[i] <- NA_character_
  }

  if (feature_id_col %in% names(ptm_rowdata_original)) {
    phosr_sequence[i] <- as.character(ptm_rowdata_original[[feature_id_col]][i])
  } else {
    phosr_sequence[i] <- NA_character_
  }

  site_info <- NA_character_
  rowname <- rownames(ptm_rowdata_original)[i]

  site_match <- regmatches(rowname, regexpr("_[STY][0-9]+", rowname))
  if (length(site_match) > 0) {
    site_str <- gsub("^_", "", site_match)
    site_info <- site_str
  }

  if (!is.na(phosr_gene[i]) && phosr_gene[i] != "" && !is.na(site_info)) {
    phosr_site_id[i] <- paste0(phosr_gene[i], ";", site_info, ";")
  } else if (!is.na(phosr_protein[i]) && !is.na(site_info)) {
    phosr_site_id[i] <- paste0(phosr_protein[i], ";", site_info, ";")
  } else {
    phosr_site_id[i] <- rowname
  }
}

ptm_rowdata_clean <- DataFrame(
  Protein = phosr_protein,
  Gene = phosr_gene,
  Sequence = phosr_sequence,
  Site_ID = phosr_site_id
)

valid_genes <- sum(!is.na(phosr_gene) & phosr_gene != "")
genes_from_fasta <- if (!is.null(accession_to_gene)) {
  sum(sapply(seq_along(phosr_protein), function(i) {
    !is.na(phosr_protein[i]) && !is.null(accession_to_gene[[phosr_protein[i]]])
  }))
} else {
  0
}
message("  - Gene symbol stats: ", valid_genes, " total, ", genes_from_fasta, " from FASTA, ",
        valid_genes - genes_from_fasta, " using protein accession as fallback")

rownames(ptm_rowdata_clean) <- rownames(ptm_rowdata_original)

if ("condition" %in% colnames(ptm_coldata) && !"Condition" %in% colnames(ptm_coldata)) {
  ptm_coldata$Condition <- ptm_coldata$condition
}

se_ptm <- SummarizedExperiment(
  assays = list(log2intensity = ptm_assay),
  rowData = ptm_rowdata_clean,
  colData = ptm_coldata
)

message("  - Created SummarizedExperiment with ", nrow(se_ptm), " phosphosites and ", ncol(se_ptm), " samples")

create_phosr_ids_for_assay <- function(pe_assay, accession_to_gene_map, protein_col = "Proteins") {
  rowdata <- rowData(pe_assay)
  rowname_vec <- rownames(rowdata)

  phosr_ids <- character(length(rowname_vec))
  genes <- character(length(rowname_vec))

  for (i in seq_along(rowname_vec)) {
    rn <- rowname_vec[i]

    protein_raw <- as.character(rowdata[[protein_col]][i])
    if (is.na(protein_raw) || protein_raw == "") {
      phosr_ids[i] <- NA_character_
      genes[i] <- NA_character_
      next
    }

    protein_list <- strsplit(protein_raw, ";")[[1]]
    main_protein <- protein_list[1]
    main_protein <- gsub("^sp\\||^tr\\|", "", main_protein)
    main_protein <- sub("\\|.*", "", main_protein)

    gene_symbol <- if (!is.null(accession_to_gene_map) && !is.null(accession_to_gene_map[[main_protein]])) {
      accession_to_gene_map[[main_protein]]
    } else {
      toupper(main_protein)
    }

    genes[i] <- gene_symbol

    site_match <- regmatches(rn, regexpr("_[STY][0-9]+", rn))
    if (length(site_match) > 0) {
      site_str <- gsub("^_", "", site_match)
      phosr_ids[i] <- paste0(gene_symbol, ";", site_str, ";")
    } else {
      if (i <= 5) {
        message("    [WARNING] Row ", i, ": No site match in rowname '", rn, "'")
      }
      phosr_ids[i] <- NA_character_
    }
  }

  list(
    phosr_ids = setNames(phosr_ids, rowname_vec),
    genes = setNames(genes, rowname_vec)
  )
}

message("\nCreating MSqRob2 to PhosR ID mappings...")

dpa_mapping <- NULL
dpa_gene_mapping <- NULL
dpu_mapping <- NULL
dpu_gene_mapping <- NULL

if (exists("all_dpa") && length(all_dpa) > 0 && peptidoformAssayName %in% names(pe)) {
  message("  - Creating DPA mappings from assay: ", peptidoformAssayName)
  dpa_maps <- create_phosr_ids_for_assay(pe[[peptidoformAssayName]], accession_to_gene, protein_col)
  dpa_mapping <- dpa_maps$phosr_ids
  dpa_gene_mapping <- dpa_maps$genes

  message("    Created ", sum(!is.na(dpa_mapping)), " valid PhosR IDs from ", length(dpa_mapping), " features")
  message("    Sample DPA mappings (first 3):")
  for (i in 1:min(3, length(dpa_mapping))) {
    message("      ", names(dpa_mapping)[i], " -> ", dpa_mapping[i], " (", dpa_gene_mapping[i], ")")
  }
}

if (exists("all_dpu") && length(all_dpu) > 0 && "peptideDPU" %in% names(pe)) {
  message("  - Creating DPU mappings from assay: peptideDPU")
  dpu_maps <- create_phosr_ids_for_assay(pe[["peptideDPU"]], accession_to_gene, protein_col)
  dpu_mapping <- dpu_maps$phosr_ids
  dpu_gene_mapping <- dpu_maps$genes

  message("    Created ", sum(!is.na(dpu_mapping)), " valid PhosR IDs from ", length(dpu_mapping), " features")
  message("    Sample DPU mappings (first 3):")
  for (i in 1:min(3, length(dpu_mapping))) {
    message("      ", names(dpu_mapping)[i], " -> ", dpu_mapping[i], " (", dpu_gene_mapping[i], ")")
  }
}

message("\nTransforming differential results to PhosR format...")

if (exists("all_dpa") && length(all_dpa) > 0 && !is.null(dpa_mapping)) {
  all_dpa_phosr <- list()
  for (comp_name in names(all_dpa)) {
    diff_df <- all_dpa[[comp_name]]
    diff_df$PhosR_ID <- dpa_mapping[diff_df$feature]
    diff_df$Gene <- dpa_gene_mapping[diff_df$feature]

    na_count <- sum(is.na(diff_df$PhosR_ID))
    if (na_count > 0) {
      message("  - DPA ", comp_name, ": ", na_count, " of ", nrow(diff_df), " features unmapped (", round(100*na_count/nrow(diff_df), 1), "%)")
      message("    First 3 unmapped features: ", paste(head(diff_df$feature[is.na(diff_df$PhosR_ID)], 3), collapse = ", "))
    }

    diff_df_valid <- diff_df[!is.na(diff_df$PhosR_ID), ]
    all_dpa_phosr[[comp_name]] <- diff_df_valid
    message("  - Transformed DPA for ", comp_name, ": ", nrow(diff_df_valid), " phosphosites")
    message("    First 3 PhosR_IDs: ", paste(head(diff_df_valid$PhosR_ID, 3), collapse = ", "))
  }
  dpa_phosr_combined <- do.call(rbind, all_dpa_phosr)
  write.table(dpa_phosr_combined, file.path(output_folder, "dpa_results_phosr_format.txt"),
              sep = "\t", row.names = FALSE, quote = FALSE)
}

if (exists("all_dpu") && length(all_dpu) > 0 && !is.null(dpu_mapping)) {
  all_dpu_phosr <- list()
  for (comp_name in names(all_dpu)) {
    diff_df <- all_dpu[[comp_name]]
    diff_df$PhosR_ID <- dpu_mapping[diff_df$feature]
    diff_df$Gene <- dpu_gene_mapping[diff_df$feature]

    na_count <- sum(is.na(diff_df$PhosR_ID))
    if (na_count > 0) {
      message("  - DPU ", comp_name, ": ", na_count, " of ", nrow(diff_df), " features unmapped (", round(100*na_count/nrow(diff_df), 1), "%)")
      message("    First 3 unmapped features: ", paste(head(diff_df$feature[is.na(diff_df$PhosR_ID)], 3), collapse = ", "))
    }

    diff_df_valid <- diff_df[!is.na(diff_df$PhosR_ID), ]
    all_dpu_phosr[[comp_name]] <- diff_df_valid
    message("  - Transformed DPU for ", comp_name, ": ", nrow(diff_df_valid), " phosphosites")
    message("    First 3 PhosR_IDs: ", paste(head(diff_df_valid$PhosR_ID, 3), collapse = ", "))
  }
  dpu_phosr_combined <- do.call(rbind, all_dpu_phosr)
  write.table(dpu_phosr_combined, file.path(output_folder, "dpu_results_phosr_format.txt"),
              sep = "\t", row.names = FALSE, quote = FALSE)
}

kinase_result <- perform_kinase_analysis(
  se_ptm = se_ptm,
  fasta_parsed = fasta_parsed,
  organism = organism,
  comparisons = comparisons,
  output_folder = output_folder,
  mat_for_kinase = NULL,
  alpha = alpha,
  lfc_threshold = lfc_threshold,
  kinase_num_motifs = kinase_num_motifs,
  top_diff_sites_heatmap = top_diff_sites_heatmap,
  top_phosphosite_heatmap = top_phosphosite_heatmap
)

if (!is.null(kinase_result)) {
  message("\nPhosR pathway and network analysis...")

  diff_results_list <- NULL
  if (exists("all_dpu_phosr") && length(all_dpu_phosr) > 0) {
    diff_results_list <- all_dpu_phosr
    message("  - Using DPU results for PhosR analysis")
  } else if (exists("all_dpa_phosr") && length(all_dpa_phosr) > 0) {
    diff_results_list <- all_dpa_phosr
    message("  - Using DPA results for PhosR analysis")
  }

  if (!is.null(diff_results_list)) {
    all_pathway_results <- list()
    all_ptmset_results <- list()
    all_network_results <- list()

    for (comp_name in names(diff_results_list)) {
      message("\n  - Processing comparison: ", comp_name)

      diff_sites <- diff_results_list[[comp_name]]

      pathway_df <- perform_pathway_enrichment(
        differential_sites = diff_sites,
        organism = organism,
        output_folder = output_folder,
        comparison_name = comp_name,
        alpha = alpha,
        lfc_threshold = lfc_threshold
      )

      if (!is.null(pathway_df)) {
        all_pathway_results[[comp_name]] <- pathway_df
      }

      ptmset_df <- perform_ptmset_enrichment(
        differential_sites = diff_sites,
        organism = organism,
        output_folder = output_folder,
        comparison_name = comp_name,
        alpha = alpha,
        lfc_threshold = lfc_threshold
      )

      if (!is.null(ptmset_df)) {
        all_ptmset_results[[comp_name]] <- ptmset_df
      }

      network_df <- analyze_kinase_substrate_network(
        ks_scores = kinase_result$kssMat$combinedScoreMatrix,
        differential_sites = diff_sites,
        output_folder = output_folder,
        comparison_name = comp_name,
        alpha = alpha,
        lfc_threshold = lfc_threshold,
        top_n = 50
      )

      if (!is.null(network_df)) {
        all_network_results[[comp_name]] <- network_df
      }
    }

    if (length(all_pathway_results) > 0) {
      pathway_combined <- do.call(rbind, all_pathway_results)
      write.table(pathway_combined, file.path(output_folder, "pathway_enrichment_all_comparisons.txt"),
                  sep = "\t", row.names = FALSE, quote = FALSE)
      message("  - Saved combined pathway enrichment results: ", nrow(pathway_combined), " pathways across ", length(all_pathway_results), " comparisons")

      generate_pathway_plots(all_pathway_results, output_folder, top_n = 20)
    }

    if (length(all_ptmset_results) > 0) {
      ptmset_combined <- do.call(rbind, all_ptmset_results)
      write.table(ptmset_combined, file.path(output_folder, "ptmset_enrichment_all_comparisons.txt"),
                  sep = "\t", row.names = FALSE, quote = FALSE)
      message("  - Saved combined PTM-SET enrichment results: ", nrow(ptmset_combined), " kinase sets across ", length(all_ptmset_results), " comparisons")

      generate_ptmset_plots(all_ptmset_results, output_folder, top_n = 20)
    }

    if (length(all_network_results) > 0) {
      network_combined <- do.call(rbind, all_network_results)
      write.table(network_combined, file.path(output_folder, "kinase_substrate_network_all_comparisons.txt"),
                  sep = "\t", row.names = FALSE, quote = FALSE)
      message("  - Saved combined kinase-substrate network: ", nrow(network_combined), " interactions across ", length(all_network_results), " comparisons")

      generate_network_plot(all_network_results, output_folder, top_n = 50)
    }

    generate_signaling_network(
      kinase_activities = kinase_result$kinase_activities,
      output_folder = output_folder,
      top_kinases = 30
    )
  }
}

message("\n=== PhosR Analysis Complete ===")
