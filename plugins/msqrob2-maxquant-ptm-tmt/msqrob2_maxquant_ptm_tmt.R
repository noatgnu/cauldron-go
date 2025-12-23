library(QFeatures)
library(msqrob2)
library(limma)
library(MsCoreUtils)
library(SummarizedExperiment)
library(Biostrings)

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
fasta_file <- params$fasta_file
output_folder <- params$output_folder
annotation_file <- params$annotation_file
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
    pe <<- addAssay(pe, assay(pe[[assayName]]), name = paste0(assayName, "Norm"))
    return(paste0(assayName, "Norm"))
  }
}

do_aggregation <- function(assayName) {
  message("  - Aggregating Peptidoforms to Proteins (method: ", summarization_method, ")...")

  if (!protein_col %in% colnames(rowData(pe[[assayName]]))) {
    available_cols <- colnames(rowData(pe[[assayName]]))
    stop("Protein column '", protein_col, "' not found in rowData. Available columns: ", paste(available_cols, collapse=", "))
  }

  # DEBUG: Check protein column values
  protein_values <- rowData(pe[[assayName]])[[protein_col]]
  message("  - DEBUG: First 5 protein values: ", paste(head(protein_values, 5), collapse=", "))
  message("  - DEBUG: Sample of unique proteins: ", paste(head(unique(protein_values), 10), collapse=", "))

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

  # DEBUG: Check resulting protein rownames
  prot_rownames <- rownames(pe[["protein"]])
  message("  - DEBUG: First 5 protein rownames after aggregation: ", paste(head(prot_rownames, 5), collapse=", "))
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

if (use_external_protein) {
  # Option A: Use external protein file - process exactly like non-PTM workflow
  message("  - Loading external protein file (global proteome from separate experiment)...")
  protein_sep <- detect_delimiter(protein_file)
  protein_data <- read.table(protein_file, sep = protein_sep, header = TRUE, check.names = FALSE)
  colnames(protein_data) <- make.names(colnames(protein_data))
  safe_prot_id <- make.names(protein_id_col)
  safe_prot_feature_id <- make.names(protein_feature_id_col)

  # Create unique peptide IDs (same as non-PTM workflow line 141-142)
  protein_unique_ids <- paste(protein_data[[safe_prot_feature_id]], protein_data[[safe_prot_id]], sep = "_")
  rownames(protein_data) <- make.unique(as.character(protein_unique_ids))

  common_samples <- intersect(colnames(pe[[peptidoformAssayName]]), colnames(protein_data))
  quant_cols_prot <- which(colnames(protein_data) %in% common_samples)

  # Create QFeatures object (same as non-PTM workflow line 145-149)
  pe_prot <- readQFeatures(
    protein_data,
    quantCols = quant_cols_prot,
    name = "peptideRaw"
  )
  colData(pe_prot) <- colData(pe)

  # Filter (same as non-PTM workflow line 175-192)
  message("  - Filtering external protein data...")
  pe_prot <- zeroIsNA(pe_prot, i = "peptideRaw")
  pe_prot <- filterNA(pe_prot, i = "peptideRaw", pNA = row_filter)

  keep_protein_prot <- !is.na(rowData(pe_prot[["peptideRaw"]])[[safe_prot_id]]) & (rowData(pe_prot[["peptideRaw"]])[[safe_prot_id]] != "")
  pe_prot <- pe_prot[keep_protein_prot, , ]

  if (remove_shared_peptides) {
    protein_counts_prot <- rowData(pe_prot[["peptideRaw"]])[[safe_prot_id]]
    shared_peptides_prot <- grepl(";", protein_counts_prot) | grepl(",", protein_counts_prot)
    pe_prot <- pe_prot[!shared_peptides_prot, , ]
  }

  currentAssayName_prot <- "peptideRaw"

  # Log transform FIRST (same as non-PTM workflow line 213-221)
  message("  - Log2 transformation of external protein data...")
  if (log2_transform) {
    pe_prot <- logTransform(pe_prot, base = 2, i = currentAssayName_prot, name = "peptideLog")
    currentAssayName_prot <- "peptideLog"
  } else {
    pe_prot <- addAssay(pe_prot, assay(pe_prot[[currentAssayName_prot]]), name = "peptideLog")
    currentAssayName_prot <- "peptideLog"
  }

  # Aggregate to proteins (same as non-PTM workflow line 245-278)
  message("  - Aggregating external peptides to proteins...")
  peptide_counts_prot <- table(rowData(pe_prot[[currentAssayName_prot]])[[safe_prot_id]])
  proteins_to_keep_prot <- names(peptide_counts_prot)[peptide_counts_prot >= filter_min_peptides]
  keep_min_pep_prot <- rowData(pe_prot[[currentAssayName_prot]])[[safe_prot_id]] %in% proteins_to_keep_prot
  pe_prot <- pe_prot[keep_min_pep_prot, , ]

  if (nrow(pe_prot[[currentAssayName_prot]]) == 0) {
    stop("No features remaining in external protein data after filtering")
  }

  message("  - ", nrow(pe_prot[[currentAssayName_prot]]), " peptides for ", length(proteins_to_keep_prot), " proteins")

  summ_fun <- switch(summarization_method,
    "robust" = MsCoreUtils::robustSummary,
    "sum" = colSums,
    "mean" = colMeans,
    "median" = matrixStats::colMedians,
    MsCoreUtils::robustSummary
  )

  pe_prot <- pe_prot[, colnames(pe_prot[[currentAssayName_prot]])]
  pe_prot <- aggregateFeatures(pe_prot, i = currentAssayName_prot, fcol = safe_prot_id, name = "protein", fun = summ_fun)

  message("  - Result: ", nrow(pe_prot[["protein"]]), " proteins")

  # Add to main pe object
  pe <- addAssay(pe, pe_prot[["protein"]], name = "protein_external")
  proteinAssayName <- "protein_external"

  message("  - External global proteome loaded: ", nrow(pe[[proteinAssayName]]), " proteins")

} else {
  # Option B: Aggregate from PTM experiment (global proteome limited to proteins with PTM sites)
  message("  - No external protein file provided")
  message("  - Estimating global proteome from PTM experiment data...")

  if (aggregation_order == "before") {
    # Aggregate raw peptidoforms, then process proteins
    message("  - Strategy: Aggregate PTM peptidoforms first, then process")
    proteinAssayName <- do_aggregation("peptidoform")

    if (impute_order == "before") {
      message("  - Imputation of proteins...")
      proteinAssayName <- do_imputation(proteinAssayName)
      message("  - Normalization of proteins...")
      proteinAssayName <- do_normalization(proteinAssayName)
    } else {
      message("  - Normalization of proteins...")
      proteinAssayName <- do_normalization(proteinAssayName)
      message("  - Imputation of proteins...")
      proteinAssayName <- do_imputation(proteinAssayName)
    }

    message("  - Log2 transformation of proteins...")
    pe <- logTransform(pe, base = 2, i = proteinAssayName, name = "proteinLog")
    proteinAssayName <- "proteinLog"

  } else {
    # Aggregate processed peptidoforms (default)
    message("  - Strategy: Process PTM peptidoforms first, then aggregate")
    proteinAssayName <- do_aggregation(peptidoformAssayName)
  }

  message("  - Global proteome estimated from PTM data: ", nrow(pe[[proteinAssayName]]), " proteins")
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
# Fit protein model
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

tryCatch({
  pe <- msqrob(object = pe, i = proteinAssayName, formula = formula, ridge = ridge_penalty, robust = robust_regression, maxitRob = max_iterations)
}, error = function(e) {
  stop("Protein model fitting failed with error: ", e$message)
})

# Check if protein model fitting succeeded
if (!"msqrobModels" %in% colnames(rowData(pe[[proteinAssayName]]))) {
  stop("Protein model fitting failed. Check if there are enough observations per condition.")
}

# Check for NULL models in protein data
protein_models <- rowData(pe[[proteinAssayName]])$msqrobModels
n_null_models <- sum(sapply(protein_models, is.null))
if (n_null_models > 0) {
  message("  - Warning: ", n_null_models, " proteins failed to fit. They will be excluded from testing.")
}

# Fit PTM model
tryCatch({
  pe <- msqrob(object = pe, i = peptidoformAssayName, formula = formula, ridge = ridge_penalty, robust = robust_regression, maxitRob = max_iterations)
}, error = function(e) {
  stop("PTM model fitting failed with error: ", e$message)
})

# Check if PTM model fitting succeeded
if (!"msqrobModels" %in% colnames(rowData(pe[[peptidoformAssayName]]))) {
  stop("PTM model fitting failed. Check if there are enough observations per condition.")
}

# Check for NULL models in PTM data
ptm_models <- rowData(pe[[peptidoformAssayName]])$msqrobModels
n_null_models_ptm <- sum(sapply(ptm_models, is.null))
if (n_null_models_ptm > 0) {
  message("  - Warning: ", n_null_models_ptm, " PTM features failed to fit. They will be excluded from testing.")
}

# Fit DPU model (PTM - Protein)
if (analysis_type %in% c("DPU", "both")) {
  # Link PTM to Protein
  # We need to map rows of peptidoform to rows of protein
  ptm_prot_ids <- rowData(pe[[peptidoformAssayName]])[[protein_col]]
  
  # For DPU, we subtract Protein expression from PTM expression
  # This requires matching samples and proteins.
  # Creating a DPU assay manually
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

protein_intensities <- cbind(
  as.data.frame(rowData(pe[[proteinAssayName]])),
  as.data.frame(assay(pe[[proteinAssayName]]))
)
list_cols <- sapply(protein_intensities, is.list)
if (any(list_cols)) { protein_intensities <- protein_intensities[, !list_cols, drop = FALSE] }
write.table(protein_intensities, file = file.path(output_folder, "protein_intensities.txt"), sep = "\t", row.names = FALSE, quote = FALSE)

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

message("[8/8] Generating quality control plots...")
pdf(file.path(output_folder, "qc_plots.pdf"), width = 14, height = 10)
par(mar=c(12, 4, 4, 2))

# Peptidoform/PTM level plots
peptidoform_matrix <- assay(pe[[peptidoformAssayName]])
if (ncol(peptidoform_matrix) > 0 && any(!is.na(peptidoform_matrix))) {
  boxplot(peptidoform_matrix, las = 2, main = "PTM/Peptidoform Intensities (Normalized)",
          ylab = "Log2 Intensity", col = rainbow(ncol(peptidoform_matrix)), cex.axis = 0.6)
}

# PCA at peptidoform level
pca_data <- t(na.omit(peptidoform_matrix))
if (nrow(pca_data) > 2 && ncol(pca_data) > 2) {
  # Remove constant/zero variance columns
  col_vars <- apply(pca_data, 2, var, na.rm = TRUE)
  pca_data <- pca_data[, col_vars > 0 & !is.na(col_vars), drop = FALSE]

  if (ncol(pca_data) > 2) {
    pca_result <- prcomp(pca_data, scale. = TRUE)
    pca_variance <- summary(pca_result)$importance[2, 1:2] * 100
    plot(pca_result$x[, 1], pca_result$x[, 2],
         col = as.numeric(colData(pe)$condition), pch = 19, cex = 2,
         xlab = paste0("PC1 (", round(pca_variance[1], 1), "%)"),
         ylab = paste0("PC2 (", round(pca_variance[2], 1), "%)"),
         main = "PCA - PTM/Peptidoform Level")
    legend("topright", legend = levels(colData(pe)$condition),
           col = 1:nlevels(colData(pe)$condition), pch = 19, cex = 0.7)
  }
}

# Protein level plots
protein_matrix <- assay(pe[[proteinAssayName]])
if (ncol(protein_matrix) > 0 && any(!is.na(protein_matrix))) {
  boxplot(protein_matrix, las = 2, main = "Protein Intensities (Global)",
          ylab = "Log2 Intensity", col = rainbow(ncol(protein_matrix)), cex.axis = 0.6)
}

# PCA at protein level
pca_data_prot <- t(na.omit(protein_matrix))
if (nrow(pca_data_prot) > 2 && ncol(pca_data_prot) > 2) {
  # Remove constant/zero variance columns
  col_vars_prot <- apply(pca_data_prot, 2, var, na.rm = TRUE)
  pca_data_prot <- pca_data_prot[, col_vars_prot > 0 & !is.na(col_vars_prot), drop = FALSE]

  if (ncol(pca_data_prot) > 2) {
    pca_result_prot <- prcomp(pca_data_prot, scale. = TRUE)
    pca_variance_prot <- summary(pca_result_prot)$importance[2, 1:2] * 100
    plot(pca_result_prot$x[, 1], pca_result_prot$x[, 2],
         col = as.numeric(colData(pe)$condition), pch = 19, cex = 2,
         xlab = paste0("PC1 (", round(pca_variance_prot[1], 1), "%)"),
         ylab = paste0("PC2 (", round(pca_variance_prot[2], 1), "%)"),
         main = "PCA - Protein Level")
    legend("topright", legend = levels(colData(pe)$condition),
           col = 1:nlevels(colData(pe)$condition), pch = 19, cex = 0.7)
  }
}

# Volcano plot - DPA
if (exists("dpa_final") && !is.null(dpa_final) && nrow(dpa_final) > 0) {
  volcano_data <- dpa_final[!is.na(dpa_final$logFC) & !is.na(dpa_final$adjPval), ]
  if (nrow(volcano_data) > 0) {
    plot(volcano_data$logFC, -log10(volcano_data$adjPval),
         pch = 20, cex = 0.5, col = ifelse(volcano_data$adjPval < alpha, "red", "gray"),
         xlab = "Log2 Fold Change", ylab = "-Log10 Adjusted P-value",
         main = "Volcano Plot - DPA (PTM Abundance)")
    abline(h = -log10(alpha), lty = 2, col = "blue")
    if (lfc_threshold > 0) {
      abline(v = c(-lfc_threshold, lfc_threshold), lty = 2, col = "blue")
    }
  }
}

# Volcano plot - DPU
if (exists("dpu_final") && !is.null(dpu_final) && nrow(dpu_final) > 0) {
  volcano_data <- dpu_final[!is.na(dpu_final$logFC) & !is.na(dpu_final$adjPval), ]
  if (nrow(volcano_data) > 0) {
    plot(volcano_data$logFC, -log10(volcano_data$adjPval),
         pch = 20, cex = 0.5, col = ifelse(volcano_data$adjPval < alpha, "red", "gray"),
         xlab = "Log2 Fold Change", ylab = "-Log10 Adjusted P-value",
         main = "Volcano Plot - DPU (PTM Usage)")
    abline(h = -log10(alpha), lty = 2, col = "blue")
    if (lfc_threshold > 0) {
      abline(v = c(-lfc_threshold, lfc_threshold), lty = 2, col = "blue")
    }
  }
}

# Volcano plot - Protein
if (exists("protein_final") && !is.null(protein_final) && nrow(protein_final) > 0) {
  volcano_data <- protein_final[!is.na(protein_final$logFC) & !is.na(protein_final$adjPval), ]
  if (nrow(volcano_data) > 0) {
    plot(volcano_data$logFC, -log10(volcano_data$adjPval),
         pch = 20, cex = 0.5, col = ifelse(volcano_data$adjPval < alpha, "red", "gray"),
         xlab = "Log2 Fold Change", ylab = "-Log10 Adjusted P-value",
         main = "Volcano Plot - Protein Level")
    abline(h = -log10(alpha), lty = 2, col = "blue")
    if (lfc_threshold > 0) {
      abline(v = c(-lfc_threshold, lfc_threshold), lty = 2, col = "blue")
    }
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
