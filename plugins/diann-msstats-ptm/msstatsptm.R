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
    if (!is.null(path)) {
        if (grepl("\\.csv$", path)) {
                  loaded_file <- read.csv(path, header = TRUE, stringsAsFactors = FALSE)
                } else if (grepl("\\.tsv$|\\.txt$", path)) {
                  loaded_file <- read.delim(path, header = TRUE, sep = "\t", stringsAsFactors = FALSE)
                } else {
                  stop(paste("Unsupported file format:", path))
                }
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
useUniquePeptide <- ifelse(is.null(params$useUniquePeptide) || params$useUniquePeptide == "true", TRUE, FALSE)
removeFewMeasurements <- ifelse(is.null(params$removeFewMeasurements) || params$removeFewMeasurements == "true", TRUE, FALSE)
removeOxidationMpeptides <- ifelse(is.null(params$removeOxidationMpeptides) || params$removeOxidationMpeptides == "true", TRUE, FALSE)
RemoveProtein_with1Feature <- ifelse(is.null(params$RemoveProtein_with1Feature) || params$RemoveProtein_with1Feature == "true", TRUE, FALSE)
MBR <- ifelse(is.null(params$MBR) || params$MBR == "true", TRUE, FALSE)
comparison_matrix_file <- params$comparison_matrix_file
moderated <- ifelse(is.null(params$moderated) || params$moderated == "true", TRUE, FALSE)

ptm_input_file <- load_file_from_path(input_file)
protein_input_file <- load_file_from_path(input_protein_file)
ptm_annotation_file <- load_file_from_path(annotation_file)
msstats_ptm_annotation_file <- NULL
if (!is.null(annotation_file)) {
  msstats_ptm_annotation_file <- ptm_annotation_file[, c("Sample", "Condition", "BioReplicate")]
  msstats_ptm_annotation_file$Run <- msstats_ptm_annotation_file$Sample
  msstats_ptm_annotation_file$Sample <- NULL
  msstats_ptm_annotation_file$Fraction <- 1
  msstats_ptm_annotation_file$TechRepMixture <- 1
  #msstats_ptm_annotation_file$Mixture <- "Mixture1"
}
protein_annotation_file <- load_file_from_path(annotation_protein_file)
if (!is.null(annotation_protein_file)) {
    msstats_protein_annotation_file <- protein_annotation_file[, c("Sample", "Condition", "BioReplicate")]
    msstats_protein_annotation_file$Run <- msstats_protein_annotation_file$Sample
    msstats_protein_annotation_file$Sample <- NULL
    msstats_protein_annotation_file$Fraction <- 1
    msstats_protein_annotation_file$TechRepMixture <- 1
    #msstats_protein_annotation_file$Mixture <- "Mixture1"
}

data <- DIANNtoMSstatsPTMFormat(
        ptm_input_file,
        ptm_annotation_file,
        input_protein = protein_input_file,
        annotation_protein = protein_annotation_file,
        fasta_path = fasta_file,
        use_unmod_peptides = FALSE,
        protein_id_col = protein_id_col,
        fasta_protein_name = fasta_protein_name,
        global_qvalue_cutoff = global_qvalue_cutoff,
        qvalue_cutoff = qvalue_cutoff,
        pg_qvalue_cutoff = pg_qvalue_cutoff,
        useUniquePeptide = useUniquePeptide,
        removeFewMeasurements = removeFewMeasurements,
        removeOxidationMpeptides = removeOxidationMpeptides,
        removeProtein_with1Feature = removeProtein_with1Feature,
        MBR = MBR,
        use_log_file = FALSE,
        append = FALSE,
        verbose = TRUE,
        log_file_path = NULL
        )

summarized <- dataSummarizationPTM(
              data,
              logTrans = 2,
              normalization = "equalizeMedians",
              normalization.PTM = "equalizeMedians",
              nameStandards = NULL,
              nameStandards.PTM = NULL,
              featureSubset = "all",
              featureSubset.PTM = "all",
              remove_uninformative_feature_outlier = FALSE,
              remove_uninformative_feature_outlier.PTM = FALSE,
              min_feature_count = 2,
              min_feature_count.PTM = 1,
              n_top_feature = 3,
              n_top_feature.PTM = 3,
              summaryMethod = "TMP",
              equalFeatureVar = TRUE,
              censoredInt = "NA",
              MBimpute = TRUE,
              MBimpute.PTM = TRUE,
              remove50missing = FALSE,
              fix_missing = NULL,
              maxQuantileforCensored = 0.999,
              use_log_file = TRUE,
              append = TRUE,
              verbose = TRUE,
              log_file_path = NULL,
              base = "MSstatsPTM_log_"
              )

if (is.null(comparison_matrix_file)) {
    comparison_matrix <- {
      if (is.null(msstats_ptm_annotation_file)) stop("ptm annotation file is NULL; cannot build comparison matrix")
      conds <- unique(as.character(msstats_ptm_annotation_file$Condition))
      if (length(conds) < 2) stop("Need at least two conditions to build pairwise comparison matrix")
      combos <- combn(conds, 2)
      rn <- apply(combos, 2, function(x) paste0(x[1], "-", x[2]))
      cm <- matrix(0, nrow = ncol(combos), ncol = length(conds),
                   dimnames = list(rn, conds))
      for (k in seq_len(ncol(combos))) {
        a <- combos[1, k]; b <- combos[2, k]
        cm[k, a] <- 1
        cm[k, b] <- -1
      }
      cm
    }
} else {
    if (grepl("\\.csv$", comparison_matrix_file)) {
          comparison_matrix <- read.csv(comparison_matrix_file, header = TRUE, stringsAsFactors = FALSE)
        } else if (grepl("\\.tsv$|\\.txt$", comparison_matrix_file)) {
          comparison_matrix <- read.delim(comparison_matrix_file, header = TRUE, sep = "\t", stringsAsFactors = FALSE)
        } else {
          stop(paste("Unsupported file format:", comparison_matrix_file))
        }
    }
}

group_comparison <- groupComparisonPTM(
    summarized,
    contrast.matrix = comparison_matrix,
    moderated = moderated,
    adj.method = "BH",
    log_base = 2,
    save_fitted_models = TRUE,
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
