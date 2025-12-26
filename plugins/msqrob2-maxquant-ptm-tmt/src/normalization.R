do_imputation <- function(pe, assayName, impute_method) {
  if (!is.null(impute_method) && impute_method != "none") {
    pe <- impute(pe, method = impute_method, i = assayName, name = paste0(assayName, "Imputed"))
    return(list(pe = pe, assayName = paste0(assayName, "Imputed")))
  }
  return(list(pe = pe, assayName = assayName))
}

do_normalization <- function(pe, assayName, normalize_method) {
  if (normalize_method != "none") {
    pe <- normalize(pe, i = assayName, name = paste0(assayName, "Norm"), method = normalize_method)
    return(list(pe = pe, assayName = paste0(assayName, "Norm")))
  } else {
    pe <- addAssay(pe, pe[[assayName]], name = paste0(assayName, "Norm"))
    return(list(pe = pe, assayName = paste0(assayName, "Norm")))
  }
}

do_aggregation <- function(pe, assayName, protein_col, filter_min_peptides, summarization_method) {
  message("  - Aggregating Peptidoforms to Proteins (method: ", summarization_method, ")...")

  if (!protein_col %in% colnames(rowData(pe[[assayName]]))) {
    available_cols <- colnames(rowData(pe[[assayName]]))
    stop("Protein column '", protein_col, "' not found in rowData. Available columns: ", paste(available_cols, collapse=", "))
  }

  protein_values <- rowData(pe[[assayName]])[[protein_col]]

  if (any(grepl("\\.[0-9]+$", protein_values))) {
    message("  - WARNING: Protein column contains numeric suffixes! Cleaning...")
    clean_proteins <- sub("\\.[0-9]+$", "", protein_values)
    rowData(pe[[assayName]])[[protein_col]] <- clean_proteins
    message("  - Cleaned protein values")
  }

  peptide_counts <- table(rowData(pe[[assayName]])[[protein_col]])
  proteins_to_keep <- names(peptide_counts)[peptide_counts >= filter_min_peptides]
  keep_min_pep <- rowData(pe[[assayName]])[[protein_col]] %in% proteins_to_keep
  pe <- pe[keep_min_pep, , ]

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

  pe <- pe[, colnames(pe[[assayName]])]
  pe <- aggregateFeatures(pe, i = assayName, fcol = protein_col, name = "protein", fun = summ_fun)

  message("  - Result: ", nrow(pe[["protein"]]), " proteins")

  return(list(pe = pe, assayName = "protein"))
}

process_ptm_peptidoforms <- function(pe, impute_order, impute_method, normalize_method, output_folder) {
  message("[5/8] Processing PTM peptidoform data...")

  currentAssayName <- "peptidoform"
  pe <- addAssay(pe, pe[[currentAssayName]], name = "peptidoform_raw")

  if (impute_order == "before") {
    message("  - Imputation (method: ", ifelse(is.null(impute_method) || impute_method == "none", "none", impute_method), ")...")
    result <- do_imputation(pe, currentAssayName, impute_method)
    pe <- result$pe
    peptidoformAssayName <- result$assayName
    message("  - Normalization (method: ", normalize_method, ")...")
    result <- do_normalization(pe, peptidoformAssayName, normalize_method)
    pe <- result$pe
    peptidoformAssayName <- result$assayName
  } else {
    message("  - Normalization (method: ", normalize_method, ")...")
    result <- do_normalization(pe, currentAssayName, normalize_method)
    pe <- result$pe
    peptidoformAssayName <- result$assayName
    message("  - Imputation (method: ", ifelse(is.null(impute_method) || impute_method == "none", "none", impute_method), ")...")
    result <- do_imputation(pe, peptidoformAssayName, impute_method)
    pe <- result$pe
    peptidoformAssayName <- result$assayName
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

  return(list(pe = pe, peptidoformAssayName = peptidoformAssayName))
}
