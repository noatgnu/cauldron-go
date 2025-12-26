generate_qc_plots <- function(se, output_folder, data_type = "PTM") {
  message("\n[8/9] Generating QC and volcano plots...")

  if (data_type == "PTM") {
    pdf_file <- file.path(output_folder, "ptm_qc_plots.pdf")
    title_prefix <- "Phosphosite"
  } else {
    pdf_file <- file.path(output_folder, "protein_qc_plots.pdf")
    title_prefix <- "Protein"
  }

  pdf(pdf_file, width = 10, height = 8)

  boxplot_data <- as.data.frame(assay(se, "log2intensity"))
  boxplot_data_long <- reshape2::melt(as.matrix(boxplot_data),
                                      varnames = c(ifelse(data_type == "PTM", "Site", "Protein"), "Sample"),
                                      value.name = "Intensity")
  boxplot_data_long$Condition <- colData(se)[as.character(boxplot_data_long$Sample), "Condition"]

  p <- ggplot(boxplot_data_long, aes(x = Sample, y = Intensity, fill = Condition)) +
    geom_boxplot() +
    theme_minimal() +
    theme(axis.text.x = element_text(angle = 45, hjust = 1)) +
    ggtitle(paste0(title_prefix, " Intensities by Sample")) +
    ylab("Log2 Intensity")
  print(p)

  pca_mat <- assay(se, "log2intensity")
  pca_mat <- pca_mat[complete.cases(pca_mat), ]
  if (nrow(pca_mat) > 0) {
    pca_res <- prcomp(t(pca_mat), scale. = FALSE)
    pca_df <- as.data.frame(pca_res$x[, 1:2])
    pca_df$Sample <- rownames(pca_df)
    pca_df$Condition <- colData(se)[pca_df$Sample, "Condition"]

    var_exp <- summary(pca_res)$importance[2, ]
    p <- ggplot(pca_df, aes(x = PC1, y = PC2, color = Condition, label = Sample)) +
      geom_point(size = 3) +
      geom_text(vjust = -0.5, size = 3) +
      theme_minimal() +
      ggtitle(paste0("PCA of ", title_prefix, " Data")) +
      xlab(paste0("PC1 (", round(var_exp[1] * 100, 1), "%)")) +
      ylab(paste0("PC2 (", round(var_exp[2] * 100, 1), "%)"))
    print(p)
  }

  cor_mat <- cor(assay(se, "log2intensity"), use = "pairwise.complete.obs")
  pheatmap::pheatmap(cor_mat, main = paste0(title_prefix, " Sample Correlation Heatmap"),
                     display_numbers = TRUE, number_format = "%.2f")

  dev.off()
  message("Saved ", data_type, " QC plots")
}

generate_volcano_plots <- function(all_results, alpha, lfc_threshold, output_folder, data_type = "PTM") {
  if (length(all_results) == 0) return()

  if (data_type == "PTM") {
    pdf_file <- file.path(output_folder, "ptm_volcano_plots.pdf")
    title_prefix <- "Volcano Plot (PTM)"
  } else {
    pdf_file <- file.path(output_folder, "protein_volcano_plots.pdf")
    title_prefix <- "Volcano Plot (Protein)"
  }

  pdf(pdf_file, width = 10, height = 8)

  for (comp_label in names(all_results)) {
    res <- all_results[[comp_label]]
    res$Significant <- (abs(res$logFC) >= lfc_threshold) & (res$adj.P.Val <= alpha)

    p <- ggplot(res, aes(x = logFC, y = -log10(adj.P.Val), color = Significant)) +
      geom_point(alpha = 0.5) +
      scale_color_manual(values = c("grey", "red")) +
      theme_minimal() +
      ggtitle(paste0(title_prefix, ": ", comp_label)) +
      xlab("Log2 Fold Change") +
      ylab("-Log10 Adjusted P-value") +
      geom_hline(yintercept = -log10(alpha), linetype = "dashed") +
      geom_vline(xintercept = c(-lfc_threshold, lfc_threshold), linetype = "dashed")
    print(p)
  }

  dev.off()
  message("Saved ", data_type, " volcano plots")
}

generate_site_class_distribution <- function(mat_for_kinase, se_ptm, output_folder) {
  tryCatch({
    pdf(file.path(output_folder, "site_class_distribution.pdf"), width = 10, height = 8)

    site_info <- data.frame(
      Site = rownames(mat_for_kinase),
      stringsAsFactors = FALSE
    )

    site_info$Residue <- sapply(strsplit(site_info$Site, ";"), function(x) {
      site_part <- x[2]
      if (!is.na(site_part) && nchar(site_part) > 0) {
        substr(site_part, 1, 1)
      } else {
        NA
      }
    })

    site_info <- site_info[!is.na(site_info$Residue), ]

    if (nrow(site_info) > 0) {
      site_counts <- as.data.frame(table(site_info$Residue))
      colnames(site_counts) <- c("Residue", "Count")
      site_counts$Percentage <- round(site_counts$Count / sum(site_counts$Count) * 100, 1)

      p <- ggplot(site_counts, aes(x = Residue, y = Count, fill = Residue)) +
        geom_bar(stat = "identity") +
        geom_text(aes(label = paste0(Count, "\n(", Percentage, "%)")), vjust = -0.5) +
        scale_fill_manual(values = c("S" = "#E74C3C", "T" = "#3498DB", "Y" = "#2ECC71")) +
        theme_minimal() +
        theme(legend.position = "none") +
        ggtitle("Distribution of Phosphorylated Residues") +
        xlab("Phosphorylated Residue") +
        ylab("Count") +
        ylim(0, max(site_counts$Count) * 1.15)
      print(p)

      if (ncol(mat_for_kinase) > 1) {
        condition_info <- data.frame(
          Sample = colnames(mat_for_kinase),
          Condition = colData(se_ptm)$Condition[match(colnames(mat_for_kinase), colnames(se_ptm))]
        )

        site_by_condition <- merge(site_info, data.frame(t(mat_for_kinase)), by.x = "Site", by.y = "row.names")
        site_by_condition_long <- reshape2::melt(site_by_condition, id.vars = c("Site", "Residue"),
                                                 variable.name = "Sample", value.name = "Intensity")
        site_by_condition_long <- merge(site_by_condition_long, condition_info, by = "Sample")
        site_by_condition_long <- site_by_condition_long[!is.na(site_by_condition_long$Intensity), ]

        if (nrow(site_by_condition_long) > 0) {
          condition_residue_counts <- aggregate(Site ~ Condition + Residue,
                                                data = site_by_condition_long,
                                                FUN = function(x) length(unique(x)))
          colnames(condition_residue_counts)[3] <- "Count"

          p2 <- ggplot(condition_residue_counts, aes(x = Condition, y = Count, fill = Residue)) +
            geom_bar(stat = "identity", position = "dodge") +
            scale_fill_manual(values = c("S" = "#E74C3C", "T" = "#3498DB", "Y" = "#2ECC71")) +
            theme_minimal() +
            theme(axis.text.x = element_text(angle = 45, hjust = 1)) +
            ggtitle("Detected Phosphosites by Residue Type per Condition") +
            xlab("Condition") +
            ylab("Number of Sites")
          print(p2)
        }
      }
    }

    dev.off()
    message("  - Saved site class distribution plots")
  }, error = function(e) {
    if (dev.cur() != 1) dev.off()
    message("  - Warning: Site class distribution plotting failed: ", e$message)
  })
}

generate_top_differential_sites_heatmap <- function(mat_for_kinase, alpha, lfc_threshold, output_folder, top_n = 50) {
  tryCatch({
    diff_sites_file <- file.path(output_folder, "dpu_results_phosr_format.txt")

    if (!file.exists(diff_sites_file)) {
      diff_sites_file <- file.path(output_folder, "dpa_results_phosr_format.txt")
    }

    message("  - Looking for PhosR-formatted differential results: ", diff_sites_file)
    message("  - File exists: ", file.exists(diff_sites_file))

    if (!file.exists(diff_sites_file)) {
      message("  - No PhosR-formatted differential results found, skipping heatmap")
      return(NULL)
    }

    pdf(file.path(output_folder, "top_differential_sites.pdf"), width = 14, height = 12)

    diff_results <- read.table(diff_sites_file, header = TRUE, sep = "\t", stringsAsFactors = FALSE)
    message("  - Loaded ", nrow(diff_results), " differential results")
    message("  - Columns: ", paste(colnames(diff_results), collapse = ", "))

    if ("adjPval" %in% colnames(diff_results) && "logFC" %in% colnames(diff_results)) {
      valid_stats <- !is.na(diff_results$adjPval) & !is.na(diff_results$logFC)
      diff_results$sig <- valid_stats & diff_results$adjPval < alpha & abs(diff_results$logFC) > lfc_threshold

      if ("PhosR_ID" %in% colnames(diff_results)) {
        diff_results$in_kinase_matrix <- diff_results$PhosR_ID %in% rownames(mat_for_kinase)
      } else {
        diff_results$in_kinase_matrix <- FALSE
      }

      message("  - Significant sites (total): ", sum(diff_results$sig, na.rm = TRUE))
      message("  - Significant sites in kinase matrix: ", sum(diff_results$sig & diff_results$in_kinase_matrix, na.rm = TRUE))

      if (sum(diff_results$sig & diff_results$in_kinase_matrix, na.rm = TRUE) > 0) {
        all_top_sig <- list()

        if ("comparison" %in% colnames(diff_results)) {
          comparisons <- unique(diff_results$comparison)
          message("  - Processing ", length(comparisons), " comparisons")

          for (comp in comparisons) {
            comp_data <- diff_results[diff_results$comparison == comp, ]
            comp_sig <- comp_data[comp_data$sig & !is.na(comp_data$sig) & comp_data$in_kinase_matrix, ]

            if (nrow(comp_sig) > 0) {
              comp_sig <- comp_sig[order(abs(comp_sig$logFC), decreasing = TRUE), ]
              top_n_actual <- min(top_n, nrow(comp_sig))
              comp_sig <- head(comp_sig, top_n_actual)
              all_top_sig[[comp]] <- comp_sig
              message("    - ", comp, ": ", nrow(comp_sig), " top sites")
            }
          }
        } else {
          comp_sig <- diff_results[diff_results$sig & !is.na(diff_results$sig) & diff_results$in_kinase_matrix, ]
          comp_sig <- comp_sig[order(abs(comp_sig$logFC), decreasing = TRUE), ]
          top_n_actual <- min(top_n, nrow(comp_sig))
          comp_sig <- head(comp_sig, top_n_actual)
          all_top_sig[[1]] <- comp_sig
        }

        top_sig <- do.call(rbind, all_top_sig)

        write.table(top_sig, file.path(output_folder, "top_differential_sites.txt"),
                    sep = "\t", row.names = FALSE, quote = FALSE)
        message("  - Saved ", nrow(top_sig), " differential sites to text file (", length(all_top_sig), " comparisons)")

        if (nrow(top_sig) > 1) {
          site_ids <- if ("PhosR_ID" %in% colnames(top_sig)) {
            top_sig$PhosR_ID
          } else if ("feature" %in% colnames(top_sig)) {
            top_sig$feature
          } else {
            rownames(top_sig)
          }

          site_ids_in_mat <- intersect(site_ids, rownames(mat_for_kinase))

          if (length(site_ids_in_mat) > 1) {
            top_mat <- mat_for_kinase[site_ids_in_mat, ]
            message("  - Drawing top differential sites heatmap with ", nrow(top_mat), " sites")

            pheatmap::pheatmap(
              top_mat,
              main = paste0("Top ", nrow(top_mat), " Differential Phosphosites (adj.P < ", alpha, ", |logFC| > ", lfc_threshold, ")"),
              scale = "row",
              cluster_rows = TRUE,
              cluster_cols = TRUE,
              fontsize_row = 6,
              fontsize_col = 9,
              color = colorRampPalette(c("blue", "white", "red"))(100)
            )
          } else {
            plot.new()
            text(0.5, 0.5, paste0("Not enough sites in matrix for heatmap\n",
                                 "Significant sites: ", length(site_ids), ", Found in matrix: ", length(site_ids_in_mat)),
                 cex = 1.2)
            message("  - Not enough sites in matrix for heatmap")
          }
        } else {
          plot.new()
          text(0.5, 0.5, paste0("Not enough significant sites for heatmap\nSignificant sites: ", nrow(top_sig)),
               cex = 1.2)
          message("  - Not enough significant sites for heatmap")
        }
      } else {
        total_sig <- sum(diff_results$sig, na.rm = TRUE)
        sig_in_matrix <- sum(diff_results$sig & diff_results$in_kinase_matrix, na.rm = TRUE)
        plot.new()
        text(0.5, 0.5, paste0("No significant sites in kinase matrix for heatmap\n",
                             "Total significant sites: ", total_sig, "\n",
                             "Significant sites with kinase data: ", sig_in_matrix, "\n",
                             "(alpha=", alpha, ", lfc_threshold=", lfc_threshold, ")"),
             cex = 1.0)
        message("  - No significant sites in kinase matrix (total sig: ", total_sig, ", in matrix: ", sig_in_matrix, ")")
      }
    } else {
      plot.new()
      text(0.5, 0.5, "Missing required columns in differential results", cex = 1.2)
      message("  - Missing required columns in differential results")
    }

    dev.off()
    message("  - Saved top differential sites heatmap")
  }, error = function(e) {
    if (dev.cur() != 1) dev.off()
    message("  - Warning: Top differential sites plotting failed: ", e$message)
  })
}

generate_kinase_family_analysis <- function(ks_activity, kinase_mat, output_folder) {
  tryCatch({
    pdf(file.path(output_folder, "kinase_family_analysis.pdf"), width = 12, height = 10)

    kinase_family_map <- list(
      "MAPK" = c("MAPK1", "MAPK3", "MAPK8", "MAPK9", "MAPK10", "MAPK11", "MAPK12", "MAPK13", "MAPK14"),
      "CDK" = c("CDK1", "CDK2", "CDK4", "CDK6", "CDK7", "CDK9"),
      "AKT" = c("AKT1", "AKT2", "AKT3"),
      "PKC" = c("PRKCA", "PRKCB", "PRKCD", "PRKCE", "PRKCG", "PRKCH", "PRKCI", "PRKCQ", "PRKCZ"),
      "PKA" = c("PRKACA", "PRKACB", "PRKACG"),
      "GSK3" = c("GSK3A", "GSK3B"),
      "SRC" = c("SRC", "FYN", "YES1", "LCK", "LYN", "HCK", "FGR", "BLK"),
      "ABL" = c("ABL1", "ABL2"),
      "AMPK" = c("PRKAA1", "PRKAA2"),
      "mTOR" = c("MTOR"),
      "ATM/ATR" = c("ATM", "ATR", "PRKDC"),
      "Aurora" = c("AURKA", "AURKB", "AURKC"),
      "PLK" = c("PLK1", "PLK2", "PLK3", "PLK4"),
      "CK2" = c("CSNK2A1", "CSNK2A2")
    )

    all_kinases <- rownames(ks_activity)
    kinase_family_df <- data.frame(
      Kinase = character(),
      Family = character(),
      stringsAsFactors = FALSE
    )

    for (family in names(kinase_family_map)) {
      family_kinases <- intersect(kinase_family_map[[family]], all_kinases)
      if (length(family_kinases) > 0) {
        kinase_family_df <- rbind(kinase_family_df,
                                  data.frame(Kinase = family_kinases,
                                             Family = family,
                                             stringsAsFactors = FALSE))
      }
    }

    if (nrow(kinase_family_df) > 0) {
      family_counts <- as.data.frame(table(kinase_family_df$Family))
      colnames(family_counts) <- c("Family", "Count")
      family_counts <- family_counts[order(family_counts$Count, decreasing = TRUE), ]

      p <- ggplot(family_counts, aes(x = reorder(Family, Count), y = Count, fill = Family)) +
        geom_bar(stat = "identity") +
        geom_text(aes(label = Count), hjust = -0.2) +
        coord_flip() +
        theme_minimal() +
        theme(legend.position = "none") +
        ggtitle("Detected Kinases by Family") +
        xlab("Kinase Family") +
        ylab("Number of Kinases Detected") +
        ylim(NA, max(family_counts$Count) * 1.1)
      print(p)

      if (ncol(kinase_mat) > 1 && nrow(kinase_family_df) > 5) {
        family_kinases_in_mat <- intersect(kinase_family_df$Kinase, rownames(kinase_mat))

        if (length(family_kinases_in_mat) > 5) {
          family_mat <- kinase_mat[family_kinases_in_mat, , drop = FALSE]
          family_annotation <- kinase_family_df[match(rownames(family_mat), kinase_family_df$Kinase), ]

          annotation_row <- data.frame(
            Family = family_annotation$Family,
            row.names = rownames(family_mat)
          )

          pheatmap::pheatmap(
            family_mat,
            main = "Kinase Activities Grouped by Family",
            scale = "row",
            cluster_rows = TRUE,
            cluster_cols = FALSE,
            annotation_row = annotation_row,
            fontsize_row = 7,
            fontsize_col = 9,
            color = colorRampPalette(c("blue", "white", "red"))(100)
          )
        }
      }
    }

    dev.off()
    message("  - Saved kinase family analysis plots")
  }, error = function(e) {
    if (dev.cur() != 1) dev.off()
    message("  - Warning: Kinase family analysis plotting failed: ", e$message)
  })
}

generate_phosphosite_heatmap <- function(kssMat, output_folder, top_n = 50) {
  tryCatch({
    pdf(file.path(output_folder, "phosphosite_heatmap.pdf"), width = 12, height = 14)

    if (!is.null(kssMat$combinedScoreMatrix) && nrow(kssMat$combinedScoreMatrix) > 0) {
      ks_scores <- kssMat$combinedScoreMatrix

      top_n_actual <- min(top_n, nrow(ks_scores))
      score_variance <- apply(ks_scores, 1, var, na.rm = TRUE)
      top_sites_idx <- order(score_variance, decreasing = TRUE)[1:top_n_actual]
      top_sites_scores <- ks_scores[top_sites_idx, ]

      if (nrow(top_sites_scores) > 1 && ncol(top_sites_scores) > 1) {
        message("  - Drawing phosphosite heatmap with ", nrow(top_sites_scores), " sites and ", ncol(top_sites_scores), " kinases")
        pheatmap::pheatmap(
          top_sites_scores,
          main = paste0("Top ", top_n_actual, " Phosphosites by Kinase Score Variance"),
          scale = "row",
          cluster_rows = TRUE,
          cluster_cols = TRUE,
          fontsize_row = 6,
          fontsize_col = 8,
          color = colorRampPalette(c("blue", "white", "red"))(100)
        )
      }
    }

    dev.off()
    message("  - Saved phosphosite heatmap")
  }, error = function(e) {
    if (dev.cur() != 1) dev.off()
    message("  - Warning: Phosphosite heatmap generation failed: ", e$message)
  })
}

generate_kinase_activity_plots <- function(ks_activity, kinase_activities, kinase_diff, output_folder) {
  tryCatch({
    kinase_mat <- as.matrix(kinase_activities[, -1, drop = FALSE])
    rownames(kinase_mat) <- kinase_activities$Kinase

    if (all(is.na(kinase_mat))) {
      message("  - Skipping heatmap: all kinase activities are NA (sample name mismatch)")
    } else if (nrow(kinase_mat) > 1 && ncol(kinase_mat) > 0 && sum(!is.na(kinase_mat)) > 0) {
      pdf(file.path(output_folder, "kinase_heatmap.pdf"), width = 12, height = 10)
      message("  - Drawing kinase activity heatmap...")
      pheatmap::pheatmap(kinase_mat, main = "Kinase Activities Across Conditions",
                         scale = "row", cluster_cols = FALSE, cluster_rows = TRUE,
                         color = colorRampPalette(c("blue", "white", "red"))(100))
      dev.off()
      message("  - Saved kinase activity heatmap")
    } else {
      message("  - Skipping heatmap: insufficient data (", nrow(kinase_mat), " kinases, ", ncol(kinase_mat), " conditions)")
    }

    if (length(kinase_diff) > 0) {
      pdf(file.path(output_folder, "kinase_differential_barplots.pdf"), width = 10, height = 8)

      for (comp_label in names(kinase_diff)) {
        diff_df <- kinase_diff[[comp_label]]
        diff_df <- diff_df[order(abs(diff_df$Difference), decreasing = TRUE), ]
        top_diff <- head(diff_df, min(30, nrow(diff_df)))

        if (nrow(top_diff) > 0) {
          p <- ggplot(top_diff, aes(x = reorder(Kinase, Difference), y = Difference, fill = Difference > 0)) +
            geom_bar(stat = "identity") +
            coord_flip() +
            scale_fill_manual(values = c("blue", "red")) +
            theme_minimal() +
            ggtitle(paste0("Top ", nrow(top_diff), " Differential Kinase Activities: ", comp_label)) +
            xlab("Kinase") +
            ylab("Activity Difference")
          print(p)
        }
      }

      dev.off()
      message("  - Saved kinase differential bar plots")
    }
  }, error = function(e) {
    if (dev.cur() != 1) dev.off()
    message("Warning: Kinase plotting failed: ", e$message)
    message("Kinase data files were saved successfully, but plots could not be generated")
  })
}
