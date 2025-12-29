perform_kinase_analysis <- function(se_ptm, fasta_parsed, organism, comparisons, output_folder,
                                    mat_for_kinase = NULL, alpha, lfc_threshold,
                                    kinase_num_motifs = 5, top_diff_sites_heatmap = 50,
                                    top_phosphosite_heatmap = 50) {
  message("\nPhosR kinase activity analysis...")

  if (is.null(fasta_parsed)) {
    message("  - Skipping kinase analysis: FASTA required for sequence window extraction")
    return(NULL)
  }

  data("PhosphoSitePlus", package = "PhosR")

  if (organism == "human") {
    ks_db <- PhosphoSite.human
  } else if (organism == "mouse") {
    ks_db <- PhosphoSite.mouse
  } else {
    stop("Unsupported organism: ", organism)
  }

  message("  - Extracting sequence windows from FASTA...")

  if (is.null(mat_for_kinase)) {
    mat_for_kinase <- assay(se_ptm, "log2intensity")
  }

  site_ids_all <- rowData(se_ptm)$Site_ID
  proteins_all <- rowData(se_ptm)$Protein
  genes_all <- if ("Gene" %in% colnames(rowData(se_ptm))) rowData(se_ptm)$Gene else NULL
  sequences_all <- rowData(se_ptm)$Sequence

  sequence_windows <- character(nrow(se_ptm))
  phosr_ids <- character(nrow(se_ptm))

  for (i in seq_len(nrow(se_ptm))) {
    site_id <- as.character(site_ids_all[i])
    protein <- as.character(proteins_all[i])

    if (is.na(site_id) || site_id == "") {
      sequence_windows[i] <- NA_character_
      phosr_ids[i] <- NA_character_
      next
    }

    phosr_ids[i] <- site_id

    if (!is.null(genes_all) && !is.na(genes_all[i]) && genes_all[i] != "") {
      gene_symbol <- genes_all[i]
    } else {
      gene_symbol <- toupper(protein)
    }

    site_parts <- strsplit(site_id, ";")[[1]]
    if (length(site_parts) >= 2 && site_parts[2] != "") {
      sites_part <- site_parts[2]
    } else {
      sequence_windows[i] <- NA_character_
      next
    }

    if (is.na(protein) || !protein %in% names(fasta_parsed)) {
      sequence_windows[i] <- NA_character_
      next
    }

    site_match <- regmatches(sites_part, regexpr("[STY][0-9]+", sites_part))
    if (length(site_match) == 0) {
      sequence_windows[i] <- NA_character_
      next
    }

    site_residue <- substr(site_match[1], 1, 1)
    site_pos <- as.numeric(substr(site_match[1], 2, nchar(site_match[1])))

    if (is.na(site_pos)) {
      sequence_windows[i] <- NA_character_
      next
    }

    prot_seq <- as.character(fasta_parsed[[protein]])
    prot_length <- nchar(prot_seq)

    window_size <- 7
    start_pos <- max(1, site_pos - window_size)
    end_pos <- min(prot_length, site_pos + window_size)

    seq_window <- substr(prot_seq, start_pos, end_pos)

    if (site_pos <= window_size) {
      padding <- paste(rep("_", window_size - site_pos + 1), collapse = "")
      seq_window <- paste0(padding, seq_window)
    }

    if (site_pos + window_size > prot_length) {
      padding <- paste(rep("_", site_pos + window_size - prot_length), collapse = "")
      seq_window <- paste0(seq_window, padding)
    }

    sequence_windows[i] <- toupper(seq_window)
  }

  phosr_ids <- gsub(";_", ";", phosr_ids)
  phosr_ids <- gsub(";;", ";", phosr_ids)

  complete_rows <- complete.cases(mat_for_kinase) & !is.na(sequence_windows)

  mat_for_kinase <- mat_for_kinase[complete_rows, , drop = FALSE]
  phosr_ids_filtered <- phosr_ids[complete_rows]
  sequence_windows_filtered <- sequence_windows[complete_rows]

  mat_for_kinase <- as.matrix(mat_for_kinase)
  storage.mode(mat_for_kinase) <- "numeric"

  duplicates <- duplicated(phosr_ids_filtered) | duplicated(phosr_ids_filtered, fromLast = TRUE)
  if (any(duplicates)) {
    n_dup <- sum(duplicated(phosr_ids_filtered))
    message("  - Aggregating ", n_dup, " duplicate phosphosites (multiple peptidoforms per site)")

    unique_ids <- unique(phosr_ids_filtered)
    mat_aggregated <- matrix(NA_real_, nrow = length(unique_ids), ncol = ncol(mat_for_kinase))
    rownames(mat_aggregated) <- unique_ids
    colnames(mat_aggregated) <- colnames(mat_for_kinase)
    seq_aggregated <- character(length(unique_ids))
    names(seq_aggregated) <- unique_ids

    for (i in seq_along(unique_ids)) {
      id <- unique_ids[i]
      rows_with_id <- which(phosr_ids_filtered == id)
      if (length(rows_with_id) == 1) {
        mat_aggregated[i, ] <- mat_for_kinase[rows_with_id, ]
        seq_aggregated[i] <- sequence_windows_filtered[rows_with_id]
      } else {
        mat_aggregated[i, ] <- colMeans(mat_for_kinase[rows_with_id, , drop = FALSE], na.rm = TRUE)
        seq_aggregated[i] <- sequence_windows_filtered[rows_with_id[1]]
      }
    }

    mat_for_kinase <- mat_aggregated
    sequence_windows_filtered <- seq_aggregated
  } else {
    rownames(mat_for_kinase) <- phosr_ids_filtered
    names(sequence_windows_filtered) <- phosr_ids_filtered
  }

  message("  - Prepared ", nrow(mat_for_kinase), " phosphosites for kinase analysis")
  message("  - Calculating kinase-substrate scores...")

  if (nrow(mat_for_kinase) != length(sequence_windows_filtered)) {
    message("ERROR: Matrix rows (", nrow(mat_for_kinase), ") != sequence windows (", length(sequence_windows_filtered), ")")
    return(NULL)
  }

  if (!is.matrix(mat_for_kinase)) {
    message("ERROR: mat_for_kinase is not a matrix!")
    return(NULL)
  }

  if (any(is.na(mat_for_kinase))) {
    message("WARNING: Matrix contains ", sum(is.na(mat_for_kinase)), " NA values")
  }

  pdf(NULL)
  dev.control('enable')

  message("  - Debug: About to call kinaseSubstrateScore")
  message("    ks_db class: ", paste(class(ks_db), collapse = ", "))
  message("    mat_for_kinase - is.matrix: ", is.matrix(mat_for_kinase), ", class: ", paste(class(mat_for_kinase), collapse = ", "))
  message("    seqs - class: ", paste(class(sequence_windows_filtered), collapse = ", "), ", length: ", length(sequence_windows_filtered))
  message("    seqs has names: ", !is.null(names(sequence_windows_filtered)))
  message("    numMotif: ", kinase_num_motifs, ", numSub: 1")

  kssMat <- tryCatch({
    kinaseSubstrateScore(
      ks_db,
      mat_for_kinase,
      seqs = sequence_windows_filtered,
      numMotif = kinase_num_motifs,
      numSub = 1
    )
  }, error = function(e) {
    message("Error in kinaseSubstrateScore: ", e$message)
    message("Full error: ", conditionMessage(e))
    return(NULL)
  }, finally = {
    if (dev.cur() != 1) dev.off()
  })

  if (is.null(kssMat)) {
    message("  - Kinase scoring returned no results")
    return(NULL)
  }

  message("  - Kinase scoring completed successfully")

  if (!is.null(kssMat$combinedScoreMatrix)) {
    ks_scores <- kssMat$combinedScoreMatrix
    write.table(ks_scores, file = file.path(output_folder, "kinase_substrate_scores.txt"),
                sep = "\t", row.names = TRUE, quote = FALSE)
    message("  - Saved kinase-substrate scores: ", nrow(ks_scores), " phosphosites x ", ncol(ks_scores), " kinases")
  } else {
    ks_scores <- NULL
  }

  message("  - Generating kinase-substrate predictions for signalome construction...")
  ks_pred <- tryCatch({
    result <- kinaseSubstratePred(kssMat, ensembleSize = 10, top = 50)
    if (!is.null(result)) {
      message("  - Kinase-substrate predictions generated successfully")
    } else {
      message("  - Warning: kinaseSubstratePred returned NULL")
    }
    result
  }, error = function(e) {
    message("  - Warning: Kinase-substrate prediction failed: ", e$message)
    return(NULL)
  })

  if (is.null(kssMat$ksActivityMatrix)) {
    message("  - ksActivityMatrix not available in kinaseSubstrateScore output")
    return(list(kssMat = kssMat, ks_scores = ks_scores, ks_pred = ks_pred, mat_for_kinase = mat_for_kinase, se_ptm = se_ptm))
  }

  ks_activity <- kssMat$ksActivityMatrix

  if (nrow(ks_activity) == 0 || ncol(ks_activity) == 0) {
    message("  - No kinases detected in the dataset")
    return(list(kssMat = kssMat, ks_scores = ks_scores, ks_pred = ks_pred, mat_for_kinase = mat_for_kinase, se_ptm = se_ptm))
  }

  kinase_activities <- data.frame(Kinase = rownames(ks_activity))

  conditions_list <- unique(colData(se_ptm)$Condition)

  for (condition in conditions_list) {
    samples_in_cond <- colnames(se_ptm)[colData(se_ptm)$Condition == condition]
    available_samples <- intersect(samples_in_cond, colnames(ks_activity))

    if (length(available_samples) > 0) {
      kinase_activities[[condition]] <- rowMeans(ks_activity[, available_samples, drop = FALSE], na.rm = TRUE)
    } else {
      message("  - Warning: No samples found for condition ", condition)
      kinase_activities[[condition]] <- NA
    }
  }

  write.table(kinase_activities, file = file.path(output_folder, "kinase_activities.txt"),
              sep = "\t", row.names = FALSE, quote = FALSE)
  message("  - Saved kinase activities")

  kinase_diff <- list()

  for (i in seq_len(nrow(comparisons))) {
    comp_label <- comparisons$comparison_label[i]
    cond_A <- comparisons$condition_A[i]
    cond_B <- comparisons$condition_B[i]

    if (cond_A %in% colnames(kinase_activities) && cond_B %in% colnames(kinase_activities)) {
      diff_df <- data.frame(
        Kinase = kinase_activities$Kinase,
        Comparison = comp_label,
        Activity_A = kinase_activities[[cond_A]],
        Activity_B = kinase_activities[[cond_B]],
        Difference = kinase_activities[[cond_A]] - kinase_activities[[cond_B]]
      )
      kinase_diff[[comp_label]] <- diff_df
    }
  }


  if (length(kinase_diff) > 0) {
    kinase_diff_all <- do.call(rbind, kinase_diff)
    write.table(kinase_diff_all, file = file.path(output_folder, "differential_kinase_activities.txt"),
                sep = "\t", row.names = FALSE, quote = FALSE)
    message("  - Saved differential kinase activities")
  }

  generate_phosphosite_heatmap(kssMat, output_folder, top_n = top_phosphosite_heatmap)
  generate_kinase_activity_plots(ks_activity, kinase_activities, kinase_diff, output_folder)
  generate_site_class_distribution(mat_for_kinase, se_ptm, output_folder)
  generate_top_differential_sites_heatmap(mat_for_kinase, alpha, lfc_threshold, output_folder, top_n = top_diff_sites_heatmap)
  generate_kinase_family_analysis(ks_activity, as.matrix(kinase_activities[, -1, drop = FALSE]), output_folder)

  return(list(
    kssMat = kssMat,
    ks_scores = ks_scores,
    ks_pred = ks_pred,
    ks_activity = ks_activity,
    kinase_activities = kinase_activities,
    kinase_diff = kinase_diff,
    mat_for_kinase = mat_for_kinase,
    se_ptm = se_ptm
  ))
}
