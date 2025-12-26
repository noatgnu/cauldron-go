load_and_prepare_ptm_data <- function(input_file, annotation_file, fasta_file = NULL,
                                      feature_id_col, protein_col, site_col, probability_col = NULL,
                                      min_probability = 0.75, col_filter = 0.7, row_filter = 0.7,
                                      exclude_conditions = NULL) {
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

  if (!is.null(fasta_file) && file.exists(fasta_file) && !is.null(probability_col)) {
    parsed_sites <- process_ptm_with_fasta(peptide_data, fasta_file, feature_id_col, protein_col, probability_col, min_probability)
    peptide_data$parsed_site <- parsed_sites
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

  keep_protein <- !is.na(rowData(pe[["peptideRaw"]])[[protein_col]])
  pe <- pe[keep_protein, , ]

  message("  - Aggregating PSMs to Peptidoforms...")
  pe <- aggregateFeatures(pe, i = "peptideRaw", fcol = "clean_feature_id", name = "peptidoform", fun = MsCoreUtils::robustSummary)

  return(list(pe = pe, protein_col = protein_col))
}
