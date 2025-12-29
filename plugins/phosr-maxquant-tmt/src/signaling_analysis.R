generate_signaling_network <- function(kinase_activities, output_folder, top_kinases = 30) {
  message("    - Generating kinase activities heatmap...")

  if (is.null(kinase_activities) || nrow(kinase_activities) == 0) {
    message("    - No kinase activities to visualize")
    return(NULL)
  }

  tryCatch({
    pdf(file.path(output_folder, "kinase_heatmap.pdf"), width = 12, height = 10)

    if (dev.cur() == 1) {
      stop("Failed to open PDF device")
    }

    kinase_mat <- as.matrix(kinase_activities[, -1, drop = FALSE])
    rownames(kinase_mat) <- kinase_activities$Kinase

    if (all(is.na(kinase_mat))) {
      message("    - Skipping heatmap: all kinase activities are NA")
      dev.off()
      return(NULL)
    }

    sig_kinases <- kinase_activities[rowSums(!is.na(kinase_mat)) > 0, ]

    if (nrow(sig_kinases) == 0) {
      message("    - No kinases with valid activity values")
      dev.off()
      return(NULL)
    }

    kinase_mat <- as.matrix(sig_kinases[, -1, drop = FALSE])
    rownames(kinase_mat) <- sig_kinases$Kinase

    activity_range <- apply(kinase_mat, 1, function(x) max(x, na.rm = TRUE) - min(x, na.rm = TRUE))
    top_kinase_names <- names(sort(activity_range, decreasing = TRUE)[1:min(top_kinases, length(activity_range))])

    if (length(top_kinase_names) > 1) {
      top_kinase_mat <- kinase_mat[top_kinase_names, , drop = FALSE]
      message("    - Drawing top ", nrow(top_kinase_mat), " variable kinases")

      pheatmap::pheatmap(
        top_kinase_mat,
        main = "Top Variable Kinases Across Conditions",
        scale = "row",
        cluster_rows = TRUE,
        cluster_cols = FALSE,
        fontsize_row = 10,
        fontsize_col = 12,
        color = colorRampPalette(c("blue", "white", "red"))(100),
        cellwidth = 50,
        cellheight = 20
      )
    } else {
      message("    - Insufficient kinases for heatmap")
    }

    dev.off()
    message("    - Saved kinase activities heatmap")
  }, error = function(e) {
    if (dev.cur() != 1) dev.off()
    message("    - Warning: Kinase heatmap failed: ", e$message)
  })
}

analyze_kinase_substrate_network <- function(ks_scores, differential_sites, output_folder, comparison_name = NULL, alpha = 0.05, lfc_threshold = 1, top_n = 50) {
  message("    - Analyzing kinase-substrate networks...")

  tryCatch({
    if (is.null(ks_scores)) {
      message("  - No kinase scores available for network analysis")
      return(NULL)
    }

    if ("adjPval" %in% colnames(differential_sites) && !"adj.P.Val" %in% colnames(differential_sites)) {
      differential_sites$adj.P.Val <- differential_sites$adjPval
    }

    valid_stats <- !is.na(differential_sites$adj.P.Val) & !is.na(differential_sites$logFC)
    sig_sites <- differential_sites[valid_stats & differential_sites$adj.P.Val < alpha & abs(differential_sites$logFC) > lfc_threshold, ]

    if (nrow(sig_sites) == 0) {
      message("  - No significant sites for network analysis")
      return(NULL)
    }

    sig_site_ids <- if ("PhosR_ID" %in% colnames(sig_sites)) sig_sites$PhosR_ID else rownames(sig_sites)

    sig_scores <- ks_scores[rownames(ks_scores) %in% sig_site_ids, , drop = FALSE]

    if (nrow(sig_scores) == 0) {
      message("    - No kinase scores for significant sites")
      return(NULL)
    }

    kinase_substrate_pairs <- list()

    for (i in seq_len(nrow(sig_scores))) {
      site <- rownames(sig_scores)[i]
      scores <- sig_scores[i, ]

      site_match <- if ("PhosR_ID" %in% colnames(sig_sites)) {
        which(sig_sites$PhosR_ID == site)
      } else {
        which(rownames(sig_sites) == site)
      }

      if (length(site_match) == 0) next

      site_idx <- site_match[1]

      top_kinases_per_site <- names(sort(scores[!is.na(scores)], decreasing = TRUE)[1:min(3, sum(!is.na(scores)))])

      for (kinase in top_kinases_per_site) {
        pair_key <- paste(kinase, site, sep = "->")
        kinase_substrate_pairs[[pair_key]] <- data.frame(
          Kinase = kinase,
          Substrate = site,
          Score = scores[kinase],
          LogFC = sig_sites$logFC[site_idx],
          AdjPValue = sig_sites$adj.P.Val[site_idx],
          stringsAsFactors = FALSE
        )
      }
    }

    if (length(kinase_substrate_pairs) > 0) {
      network_df <- do.call(rbind, kinase_substrate_pairs)
      network_df <- network_df[order(-network_df$Score), ]

      message("    - Kinase-substrate network: ", nrow(network_df), " interactions for ", comparison_name)

      return(network_df)
    }

    return(NULL)
  }, error = function(e) {
    message("  - Warning: Kinase-substrate network analysis failed: ", e$message)
    return(NULL)
  })
}

generate_network_plot <- function(all_network_results, output_folder, top_n = 50) {
  if (is.null(all_network_results) || length(all_network_results) == 0) {
    return(NULL)
  }

  tryCatch({
    pdf(file.path(output_folder, "kinase_substrate_network.pdf"), width = 14, height = 10)

    for (comp_name in names(all_network_results)) {
      network_df <- all_network_results[[comp_name]]

      if (is.null(network_df) || nrow(network_df) == 0) next

      top_interactions <- head(network_df[order(-network_df$Score), ], top_n)

      kinase_counts <- as.data.frame(table(top_interactions$Kinase))
      colnames(kinase_counts) <- c("Kinase", "SubstrateCount")
      kinase_counts <- kinase_counts[order(-kinase_counts$SubstrateCount), ]

      if (nrow(kinase_counts) > 0) {
        p <- ggplot(head(kinase_counts, 20), aes(x = reorder(Kinase, SubstrateCount), y = SubstrateCount, fill = SubstrateCount)) +
          geom_bar(stat = "identity") +
          coord_flip() +
          scale_fill_gradient(low = "lightblue", high = "darkblue") +
          theme_minimal() +
          ggtitle(paste0(comp_name, ": Top Kinases by Substrate Count (Top ", top_n, " Interactions)")) +
          xlab("Kinase") +
          ylab("Number of Significant Substrates") +
          theme(legend.position = "none")
        print(p)
      }

      top_kinases <- head(kinase_counts$Kinase, 10)
      filtered_interactions <- top_interactions[top_interactions$Kinase %in% top_kinases, ]

      if (nrow(filtered_interactions) > 0) {
        filtered_interactions$LogFCDirection <- ifelse(filtered_interactions$LogFC > 0, "Up", "Down")

        p2 <- ggplot(filtered_interactions, aes(x = Kinase, y = Score, fill = LogFCDirection)) +
          geom_boxplot() +
          scale_fill_manual(values = c("Up" = "red", "Down" = "blue")) +
          theme_minimal() +
          theme(axis.text.x = element_text(angle = 45, hjust = 1)) +
          ggtitle(paste0(comp_name, ": Kinase-Substrate Scores by Regulation Direction")) +
          xlab("Kinase") +
          ylab("Kinase-Substrate Score") +
          labs(fill = "Substrate Direction")
        print(p2)
      }
    }

    dev.off()
    message("    - Saved kinase-substrate network plots (", length(all_network_results), " comparisons)")
  }, error = function(e) {
    if (dev.cur() != 1) dev.off()
    message("  - Warning: Network plotting failed: ", e$message)
  })
}

generate_phosr_signalome_network_single <- function(kssMat, ks_pred, se_ptm, comparison_name, top_kinases = 5) {
  message("    - Generating PhosR signalome for ", comparison_name)

  if (is.null(kssMat) || is.null(ks_pred) || is.null(se_ptm)) {
    message("    - Missing data for signalome construction (need kinase scores, predictions, and phospho data)")
    return(NULL)
  }

  tryCatch({
    if (is.null(kssMat$combinedScoreMatrix)) {
      message("    - No kinase score matrix available")
      return(NULL)
    }

    kinase_scores_sums <- sort(apply(kssMat$combinedScoreMatrix, 2, function(x) sum(x, na.rm = TRUE)), decreasing = TRUE)
    kinase_OI <- head(names(kinase_scores_sums), min(top_kinases, length(kinase_scores_sums)))

    if (length(kinase_OI) == 0) {
      message("    - No kinases available for signalome construction")
      return(NULL)
    }

    message("    - Constructing signalome for top ", length(kinase_OI), " kinases")

    if (!is(se_ptm, "SummarizedExperiment")) {
      message("    - ERROR: se_ptm is not a SummarizedExperiment object")
      return(NULL)
    }

    exprsMat <- SummarizedExperiment::assay(se_ptm, "log2intensity")

    message("    - PhosR Signalomes function is unreliable - skipping signalome-based visualization")
    message("    - The kinase activities and kinase-substrate networks are available in the data outputs")
    return(NULL)
  }, error = function(e) {
    message("    - Warning: Signalome generation failed for ", comparison_name, ": ", e$message)
    return(NULL)
  })
}
