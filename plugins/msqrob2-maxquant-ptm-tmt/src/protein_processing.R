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

process_global_protein_data <- function(pe, protein_file, protein_id_col, protein_feature_id_col,
                                        annotation_protein_file, protein_col,
                                        log2_transform, aggregation_order, impute_order, impute_method,
                                        normalize_method, summarization_method, filter_min_peptides,
                                        col_filter, row_filter, remove_shared_peptides, output_folder) {
  message("[6/8] Processing global protein data...")

  use_external_protein <- !is.null(protein_file) && file.exists(protein_file)

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
       annotation_prot <- as.data.frame(colData(pe))
       annotation_prot$Sample <- rownames(annotation_prot)
       annotation_prot$Condition <- annotation_prot$condition
       if ("biorep" %in% colnames(annotation_prot)) annotation_prot$BioReplicate <- annotation_prot$biorep
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

  return(list(pe = pe, proteinAssayName = proteinAssayName))
}
