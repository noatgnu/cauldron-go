load_and_prepare_ptm_data <- function(input_file, annotation_file, feature_id_col, site_col,
                                      protein_col, probability_col, min_probability, fasta_file,
                                      exclude_conditions, log2_transform) {
  message("\n[1/9] Reading PTM data...")
  input_sep <- detect_delimiter(input_file)
  peptide_data <- read.table(input_file, sep = input_sep, header = TRUE,
                            na.strings = c("NA", "NaN", "N/A", "#VALUE!", ""),
                            check.names = FALSE, stringsAsFactors = FALSE)

  message("[2/9] Reading annotation...")
  annot_sep <- detect_delimiter(annotation_file)
  annotation <- read.table(annotation_file, sep = annot_sep, header = TRUE,
                          stringsAsFactors = FALSE, check.names = FALSE)

  colnames(peptide_data) <- make.names(colnames(peptide_data))
  annotation$Sample <- make.names(annotation$Sample)
  annotation$Condition <- make.names(annotation$Condition)
  if ("BioReplicate" %in% colnames(annotation)) annotation$BioReplicate <- make.names(annotation$BioReplicate)
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

  annotation <- annotation[annotation$Sample %in% sample_cols_in_data, ]

  if (!is.null(exclude_conditions) && exclude_conditions != "") {
    exclude_list <- trimws(unlist(strsplit(exclude_conditions, ",")))
    annotation <- annotation[!annotation$Condition %in% exclude_list, ]
    message("Excluded conditions: ", paste(exclude_list, collapse = ", "))
    message("Samples after exclusion: ", nrow(annotation))
  }

  if (nrow(annotation) == 0) {
    stop("No samples remaining after filtering")
  }

  message("\n[3/9] Processing PTM sites...")

  if ("Reverse" %in% colnames(peptide_data)) {
    peptide_data <- peptide_data[is.na(peptide_data$Reverse) | peptide_data$Reverse != "+", ]
  }
  if ("Potential.contaminant" %in% colnames(peptide_data)) {
    peptide_data <- peptide_data[is.na(peptide_data$Potential.contaminant) | peptide_data$Potential.contaminant != "+", ]
  }

  if (!is.null(fasta_file) && file.exists(fasta_file) && !is.null(probability_col) && probability_col %in% colnames(peptide_data)) {
    message("Using FASTA file to parse phosphosites with probability threshold ", min_probability)
    new_site_column <- process_ptm_with_fasta(peptide_data, fasta_file, feature_id_col, protein_col, probability_col, min_probability)
    peptide_data$Site_ID <- new_site_column
  } else {
    message("Using simple site annotation...")
    peptide_data$Site_ID <- paste0(peptide_data[[protein_col]], "_", peptide_data[[feature_id_col]])
  }

  peptide_data <- peptide_data[!is.na(peptide_data$Site_ID), ]
  message("Retained ", nrow(peptide_data), " modified PSMs")

  if (log2_transform) {
    message("Applying log2 transformation...")
    for (col in annotation$Sample) {
      if (col %in% colnames(peptide_data)) {
        peptide_data[[col]] <- log2(pmax(peptide_data[[col]], 1))
      }
    }
  }

  peptide_data_subset <- peptide_data[, c("Site_ID", protein_col, feature_id_col, annotation$Sample)]
  colnames(peptide_data_subset)[2:3] <- c("Protein", "Sequence")

  site_data <- aggregate(. ~ Site_ID + Protein + Sequence,
                        data = peptide_data_subset,
                        FUN = function(x) mean(x, na.rm = TRUE))

  intensity_mat <- as.matrix(site_data[, annotation$Sample, drop = FALSE])
  rownames(intensity_mat) <- site_data$Site_ID

  message("Aggregated to ", nrow(intensity_mat), " unique phosphosites")

  return(list(
    intensity_mat = intensity_mat,
    site_data = site_data,
    annotation = annotation,
    protein_col = protein_col,
    feature_id_col = feature_id_col,
    probability_col = probability_col
  ))
}

load_and_prepare_protein_data <- function(protein_file, protein_feature_id_col, protein_id_col,
                                          annotation_protein_file, se_ptm, log2_transform,
                                          summarization_method = "mean", aggregation_order = "after",
                                          impute = "knn", impute_order = "after", normalize_method = "center.median",
                                          col_filter = 0.7, row_filter = 0.7, remove_shared_peptides = FALSE,
                                          filter_min_peptides = 1) {
  message("\n[6/9] Processing global protein data...")

  if (is.null(protein_file) || !file.exists(protein_file)) {
    message("  - No external protein file provided. Skipping protein processing.")
    return(NULL)
  }

  message("  - Loading external protein file...")
  protein_sep <- detect_delimiter(protein_file)
  protein_data <- read.table(protein_file, sep = protein_sep, header = TRUE,
                            na.strings = c("NA", "NaN", "N/A", "#VALUE!", ""),
                            check.names = FALSE, stringsAsFactors = FALSE)

  colnames(protein_data) <- make.names(colnames(protein_data))
  protein_feature_id_col <- make.names(protein_feature_id_col)
  protein_id_col <- make.names(protein_id_col)

  if (!is.null(annotation_protein_file) && file.exists(annotation_protein_file)) {
    message("  - Loading protein annotation file...")
    annot_prot_sep <- detect_delimiter(annotation_protein_file)
    annotation_prot <- read.table(annotation_protein_file, sep = annot_prot_sep, header = TRUE,
                                  stringsAsFactors = FALSE, check.names = FALSE)
    annotation_prot$Sample <- make.names(annotation_prot$Sample)
    annotation_prot$Condition <- make.names(annotation_prot$Condition)
    if ("BioReplicate" %in% colnames(annotation_prot)) annotation_prot$BioReplicate <- make.names(annotation_prot$BioReplicate)
  } else {
    message("  - No protein annotation provided, using same as PTM...")
    annotation_prot <- as.data.frame(colData(se_ptm))
  }

  if ("Reverse" %in% colnames(protein_data)) {
    protein_data <- protein_data[is.na(protein_data$Reverse) | protein_data$Reverse != "+", ]
  }
  if ("Potential.contaminant" %in% colnames(protein_data)) {
    protein_data <- protein_data[is.na(protein_data$Potential.contaminant) | protein_data$Potential.contaminant != "+", ]
  }

  ptm_annot <- as.data.frame(colData(se_ptm))
  ptm_annot$sample <- rownames(ptm_annot)
  if (!"BioReplicate" %in% colnames(ptm_annot)) ptm_annot$BioReplicate <- ptm_annot$Sample
  if (!"BioReplicate" %in% colnames(annotation_prot)) annotation_prot$BioReplicate <- annotation_prot$Sample

  ptm_keys <- paste(ptm_annot$Condition, ptm_annot$BioReplicate, sep = "_")
  prot_keys <- paste(annotation_prot$Condition, annotation_prot$BioReplicate, sep = "_")

  common_keys <- intersect(ptm_keys, prot_keys)
  if (length(common_keys) == 0) {
    message("  - Warning: No matching samples between PTM and Protein. Skipping protein processing.")
    return(NULL)
  }

  final_ptm_samples <- ptm_annot$sample[match(common_keys, ptm_keys)]
  final_prot_samples <- annotation_prot$Sample[match(common_keys, prot_keys)]

  valid_indices <- final_prot_samples %in% colnames(protein_data)
  final_ptm_samples <- final_ptm_samples[valid_indices]
  final_prot_samples <- final_prot_samples[valid_indices]

  message("  - Matched ", length(final_ptm_samples), " samples between experiments")

  protein_data_subset <- protein_data[, c(protein_id_col, protein_feature_id_col, final_prot_samples)]

  protein_unique_ids <- paste(protein_data_subset[[protein_feature_id_col]],
                              protein_data_subset[[protein_id_col]],
                              sep = "_")
  rownames(protein_data_subset) <- make.unique(as.character(protein_unique_ids))

  numeric_cols <- final_prot_samples[final_prot_samples %in% colnames(protein_data_subset)]

  if (log2_transform) {
    for (col in numeric_cols) {
      protein_data_subset[[col]] <- log2(pmax(protein_data_subset[[col]], 1))
    }
  }

  protein_data_subset$protein_id_for_agg <- protein_data_subset[[protein_id_col]]

  agg_func <- switch(summarization_method,
    "mean" = function(x) mean(x, na.rm = TRUE),
    "median" = function(x) median(x, na.rm = TRUE),
    "sum" = function(x) sum(x, na.rm = TRUE),
    function(x) mean(x, na.rm = TRUE)
  )

  agg_formula <- as.formula(paste(paste(numeric_cols, collapse = " + "), "~ protein_id_for_agg"))
  protein_agg <- aggregate(agg_formula, data = protein_data_subset, FUN = agg_func)

  protein_intensity_mat <- as.matrix(protein_agg[, numeric_cols, drop = FALSE])
  rownames(protein_intensity_mat) <- protein_agg$protein_id_for_agg
  colnames(protein_intensity_mat) <- final_ptm_samples

  protein_rowdata <- DataFrame(
    Protein = protein_agg$protein_id_for_agg
  )

  protein_coldata <- colData(se_ptm)[final_ptm_samples, , drop = FALSE]

  se_protein <- SummarizedExperiment(
    assays = list(log2intensity = protein_intensity_mat),
    rowData = protein_rowdata,
    colData = protein_coldata
  )

  return(se_protein)
}

create_se_from_data <- function(intensity_mat, site_data, annotation) {
  message("\n[4/9] Creating SummarizedExperiment for PTM...")

  rowdata <- DataFrame(
    Protein = site_data$Protein,
    Sequence = site_data$Sequence,
    Site_ID = site_data$Site_ID
  )

  coldata <- DataFrame(annotation)
  rownames(coldata) <- annotation$Sample

  se_ptm <- SummarizedExperiment(
    assays = list(log2intensity = intensity_mat),
    rowData = rowdata,
    colData = coldata
  )

  return(se_ptm)
}
