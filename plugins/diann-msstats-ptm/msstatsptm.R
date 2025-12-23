library(MSstatsPTM)

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

write_table_safe <- function(obj, name) {
  if (is.null(obj)) {
    warning(paste(name, "is NULL; skipping write"))
    return(invisible(NULL))
  }
  fname <- file.path(output_folder, paste0("group_comparison_", tolower(name), ".txt"))
  tryCatch({
    write.table(obj, file = fname, sep = "\t", row.names = FALSE, quote = FALSE)
  }, error = function(e) {
    warning(paste("Failed to write", fname, ":", e$message))
  })
}

load_file_from_path <- function(path) {
    loaded_file <- NULL
    if (!is.null(path) && path != "") {
        if (grepl("\\.csv$", path)) {
                  loaded_file <- read.csv(path, header = TRUE, stringsAsFactors = FALSE)
                } else if (grepl("\\.tsv$|\\.txt$", path)) {
                  loaded_file <- read.delim(path, header = TRUE, sep = "\t", stringsAsFactors = FALSE)
                } else {
                  stop(paste("Unsupported file format:", path))
                }
            }
    loaded_file
}

params <- parse_args(args)
input_file <- params$input_file
annotation_file <- params$annotation_file
input_protein_file <- params$input_protein_file
annotation_protein_file <- params$annotation_protein_file
output_folder <- params$output_folder
fasta_file <- ifelse(is.null(params$fasta_file), "", params$fasta_file)
protein_id_col <- ifelse(is.null(params$protein_id_col), "Protein.Group", params$protein_id_col)
fasta_protein_name <- ifelse(is.null(params$fasta_protein_name), "uniprot_ac", params$fasta_protein_name)

global_qvalue_cutoff <- ifelse(is.null(params$global_qvalue_cutoff), 0.01, as.numeric(params$global_qvalue_cutoff))
qvalue_cutoff <- ifelse(is.null(params$qvalue_cutoff), 0.01, as.numeric(params$qvalue_cutoff))
pg_qvalue_cutoff <- ifelse(is.null(params$pg_qvalue_cutoff), 0.01, as.numeric(params$pg_qvalue_cutoff))
localization_cutoff <- ifelse(is.null(params$localization_cutoff), 0.75, as.numeric(params$localization_cutoff))
useUniquePeptide <- ifelse(is.null(params$useUniquePeptide) || params$useUniquePeptide == "true", TRUE, FALSE)
removeFewMeasurements <- ifelse(is.null(params$removeFewMeasurements) || params$removeFewMeasurements == "true", TRUE, FALSE)
removeOxidationMpeptides <- ifelse(is.null(params$removeOxidationMpeptides) || params$removeOxidationMpeptides == "true", TRUE, FALSE)
RemoveProtein_with1Feature <- ifelse(is.null(params$RemoveProtein_with1Feature) || params$RemoveProtein_with1Feature == "true", TRUE, FALSE)
use_unmod_peptides <- ifelse(is.null(params$use_unmod_peptides) || params$use_unmod_peptides == "false", FALSE, TRUE)
MBR <- ifelse(is.null(params$MBR) || params$MBR == "true", TRUE, FALSE)

logTrans <- ifelse(is.null(params$logTrans), 2, as.numeric(params$logTrans))
normalization <- ifelse(is.null(params$normalization), "equalizeMedians", params$normalization)
normalization_PTM <- ifelse(is.null(params$normalization_PTM), "equalizeMedians", params$normalization_PTM)
featureSubset <- ifelse(is.null(params$featureSubset), "all", params$featureSubset)
featureSubset_PTM <- ifelse(is.null(params$featureSubset_PTM), "all", params$featureSubset_PTM)
remove_uninformative_feature_outlier <- ifelse(is.null(params$remove_uninformative_feature_outlier) || params$remove_uninformative_feature_outlier == "false", FALSE, TRUE)
remove_uninformative_feature_outlier_PTM <- ifelse(is.null(params$remove_uninformative_feature_outlier_PTM) || params$remove_uninformative_feature_outlier_PTM == "false", FALSE, TRUE)
min_feature_count <- ifelse(is.null(params$min_feature_count), 2, as.numeric(params$min_feature_count))
min_feature_count_PTM <- ifelse(is.null(params$min_feature_count_PTM), 1, as.numeric(params$min_feature_count_PTM))
n_top_feature <- ifelse(is.null(params$n_top_feature), 3, as.numeric(params$n_top_feature))
n_top_feature_PTM <- ifelse(is.null(params$n_top_feature_PTM), 3, as.numeric(params$n_top_feature_PTM))
summaryMethod <- ifelse(is.null(params$summaryMethod), "TMP", params$summaryMethod)
equalFeatureVar <- ifelse(is.null(params$equalFeatureVar) || params$equalFeatureVar == "true", TRUE, FALSE)
censoredInt <- ifelse(is.null(params$censoredInt), "NA", params$censoredInt)
MBimpute <- ifelse(is.null(params$MBimpute) || params$MBimpute == "true", TRUE, FALSE)
MBimpute_PTM <- ifelse(is.null(params$MBimpute_PTM) || params$MBimpute_PTM == "true", TRUE, FALSE)
maxQuantileforCensored <- ifelse(is.null(params$maxQuantileforCensored), 0.999, as.numeric(params$maxQuantileforCensored))

comparison_matrix_file <- params$comparison_matrix_file
moderated <- ifelse(is.null(params$moderated) || params$moderated == "true", TRUE, FALSE)
adj_method <- ifelse(is.null(params$adj_method), "BH", params$adj_method)
log_base <- ifelse(is.null(params$log_base), 2, as.numeric(params$log_base))
save_fitted_models <- ifelse(is.null(params$save_fitted_models) || params$save_fitted_models == "true", TRUE, FALSE)

ptm_input_file <- load_file_from_path(input_file)
protein_input_file <- load_file_from_path(input_protein_file)
ptm_annotation_file <- load_file_from_path(annotation_file)
protein_annotation_file <- load_file_from_path(annotation_protein_file)

cat("Converting DIA-NN data to MSstatsPTM format...\n")
data <- DIANNtoMSstatsPTMFormat(
        input = ptm_input_file,
        annotation = ptm_annotation_file,
        input_protein = protein_input_file,
        annotation_protein = protein_annotation_file,
        fasta_path = fasta_file,
        fasta_protein_name = fasta_protein_name,
        protein_id_col = protein_id_col,
        global_qvalue_cutoff = global_qvalue_cutoff,
        qvalue_cutoff = qvalue_cutoff,
        pg_qvalue_cutoff = pg_qvalue_cutoff,
        localization_cutoff = localization_cutoff,
        useUniquePeptide = useUniquePeptide,
        removeFewMeasurements = removeFewMeasurements,
        removeOxidationMpeptides = removeOxidationMpeptides,
        removeProtein_with1Feature = RemoveProtein_with1Feature,
        use_unmod_peptides = use_unmod_peptides,
        MBR = MBR,
        labeling_type = "LF",
        use_log_file = TRUE,
        append = FALSE,
        verbose = TRUE,
        log_file_path = NULL
        )

cat("Checking converted data structure...\n")
if (!is.null(data) && length(data) > 0) {
    if (!is.null(data$PTM)) {
        cat(paste("PTM data rows:", nrow(data$PTM), "\n"))
    }
    if (!is.null(data$PROTEIN)) {
        cat(paste("PROTEIN data rows:", nrow(data$PROTEIN), "\n"))
    }
}

summarized <- dataSummarizationPTM(
              data,
              logTrans = logTrans,
              normalization = normalization,
              normalization.PTM = normalization_PTM,
              nameStandards = NULL,
              nameStandards.PTM = NULL,
              featureSubset = featureSubset,
              featureSubset.PTM = featureSubset_PTM,
              remove_uninformative_feature_outlier = remove_uninformative_feature_outlier,
              remove_uninformative_feature_outlier.PTM = remove_uninformative_feature_outlier_PTM,
              min_feature_count = min_feature_count,
              min_feature_count.PTM = min_feature_count_PTM,
              n_top_feature = n_top_feature,
              n_top_feature.PTM = n_top_feature_PTM,
              summaryMethod = summaryMethod,
              equalFeatureVar = equalFeatureVar,
              censoredInt = censoredInt,
              MBimpute = MBimpute,
              MBimpute.PTM = MBimpute_PTM,
              remove50missing = FALSE,
              fix_missing = NULL,
              maxQuantileforCensored = maxQuantileforCensored,
              use_log_file = TRUE,
              append = TRUE,
              verbose = TRUE,
              log_file_path = NULL,
              base = "MSstatsPTM_log_"
              )

cat("Checking summarized data structure...\n")
if (!is.null(summarized)) {
    if (!is.null(summarized$PTM)) {
        cat(paste("Summarized PTM rows:", nrow(summarized$PTM), "\n"))
    }
    if (!is.null(summarized$PROTEIN)) {
        cat(paste("Summarized PROTEIN rows:", nrow(summarized$PROTEIN), "\n"))
    }
}

cat("Creating wide-format intensity tables...\n")
create_wide_intensity_table <- function(data, id_col) {
    if (is.null(data)) return(NULL)

    library(tidyr)
    library(dplyr)

    intensity_col <- if ("LogIntensities" %in% names(data)) "LogIntensities" else if ("Abundance" %in% names(data)) "Abundance" else NULL

    if (is.null(intensity_col)) {
        warning("No intensity column found in summarized data")
        return(NULL)
    }

    wide_data <- data %>%
        select(all_of(c(id_col, "Run", "Condition", "BioReplicate", intensity_col))) %>%
        mutate(sample_id = paste(Condition, BioReplicate, sep = "_Rep")) %>%
        select(all_of(c(id_col, "sample_id", intensity_col))) %>%
        distinct() %>%
        pivot_wider(
            id_cols = all_of(id_col),
            names_from = sample_id,
            values_from = all_of(intensity_col),
            values_fn = mean
        )

    return(wide_data)
}

peptide_intensity_wide <- NULL
protein_intensity_wide <- NULL

if (!is.null(summarized$PTM)) {
    tryCatch({
        peptide_intensity_wide <- create_wide_intensity_table(summarized$PTM, "Peptide")
        if (!is.null(peptide_intensity_wide)) {
            cat(paste("Created peptide intensity table:", nrow(peptide_intensity_wide), "peptides x", ncol(peptide_intensity_wide) - 1, "samples\n"))
        }
    }, error = function(e) {
        warning(paste("Failed to create peptide intensity table:", e$message))
    })
}

if (!is.null(summarized$PROTEIN)) {
    tryCatch({
        protein_intensity_wide <- create_wide_intensity_table(summarized$PROTEIN, "Protein")
        if (!is.null(protein_intensity_wide)) {
            cat(paste("Created protein intensity table:", nrow(protein_intensity_wide), "proteins x", ncol(protein_intensity_wide) - 1, "samples\n"))
        }
    }, error = function(e) {
        warning(paste("Failed to create protein intensity table:", e$message))
    })
}

if (is.null(comparison_matrix_file) || comparison_matrix_file == "") {
    comparison_matrix <- "pairwise"
    cat("Using pairwise comparisons for all conditions\n")
} else {
    if (grepl("\\.csv$", comparison_matrix_file)) {
          comparison_matrix <- read.csv(comparison_matrix_file, header = TRUE, stringsAsFactors = FALSE)
        } else if (grepl("\\.tsv$|\\.txt$", comparison_matrix_file)) {
          comparison_matrix <- read.delim(comparison_matrix_file, header = TRUE, sep = "\t", stringsAsFactors = FALSE)
        } else {
          stop(paste("Unsupported file format:", comparison_matrix_file))
        }
    cat("Loaded custom comparison matrix\n")
}

group_comparison <- groupComparisonPTM(
    summarized,
    contrast.matrix = comparison_matrix,
    moderated = moderated,
    adj.method = adj_method,
    log_base = log_base,
    save_fitted_models = save_fitted_models,
    use_log_file = FALSE,
    append = FALSE,
    verbose = TRUE,
    log_file_path = NULL,
    base = "MSstatsPTM_log_",
    ptm_label_type = "LF",
    protein_label_type = "LF"
)

if (is.null(output_folder) || output_folder == "") output_folder <- "."
dir.create(output_folder, recursive = TRUE, showWarnings = FALSE)

write_table_safe(group_comparison$PTM, "PTM")
write_table_safe(group_comparison$PROTEIN, "PROTEIN")
write_table_safe(group_comparison$ADJUSTED, "ADJUSTED")

if (!is.null(peptide_intensity_wide)) {
    fname <- file.path(output_folder, "peptide_intensities_wide.txt")
    tryCatch({
        write.table(peptide_intensity_wide, file = fname, sep = "\t", row.names = FALSE, quote = FALSE)
        cat(paste("Saved peptide intensity table to:", fname, "\n"))
    }, error = function(e) {
        warning(paste("Failed to write peptide intensity table:", e$message))
    })
}

if (!is.null(protein_intensity_wide)) {
    fname <- file.path(output_folder, "protein_intensities_wide.txt")
    tryCatch({
        write.table(protein_intensity_wide, file = fname, sep = "\t", row.names = FALSE, quote = FALSE)
        cat(paste("Saved protein intensity table to:", fname, "\n"))
    }, error = function(e) {
        warning(paste("Failed to write protein intensity table:", e$message))
    })
}

cat("MSstatsPTM label-free analysis completed successfully\n")
cat(paste("Results saved to:", output_folder, "\n"))
