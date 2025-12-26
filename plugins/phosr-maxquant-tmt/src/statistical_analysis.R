perform_differential_analysis <- function(se, comparison_file, adjust_method, alpha, lfc_threshold, output_folder, data_type = "PTM") {
  if (data_type == "DPA") {
    message("  - DPA (Differential Phosphorylation Abundance)")
  } else if (data_type == "DPU") {
    message("  - DPU (Differential PTM, protein Unchanged)")
  } else if (data_type == "Protein") {
    message("  - Protein differential analysis")
  } else {
    message("\n[7/9] Statistical testing for ", data_type, "...")
  }

  if (!is.null(comparison_file) && file.exists(comparison_file)) {
    comp_sep <- detect_delimiter(comparison_file)
    comparisons <- read.table(comparison_file, sep = comp_sep, header = TRUE, stringsAsFactors = FALSE)
  } else {
    conditions <- unique(colData(se)$Condition)
    comparisons <- expand.grid(condition_A = conditions, condition_B = conditions, stringsAsFactors = FALSE)
    comparisons <- comparisons[comparisons$condition_A != comparisons$condition_B, ]
    comparisons$comparison_label <- paste0(comparisons$condition_A, "_vs_", comparisons$condition_B)
  }

  design_formula <- ~ 0 + Condition
  design_mat <- model.matrix(design_formula, data = colData(se))

  fit <- limma::lmFit(assay(se, "log2intensity"), design_mat)

  all_results <- list()
  available_conditions <- unique(colData(se)$Condition)
  valid_params <- paste0("Condition", available_conditions)

  for (i in seq_len(nrow(comparisons))) {
    comp_label <- comparisons$comparison_label[i]
    cond_A <- comparisons$condition_A[i]
    cond_B <- comparisons$condition_B[i]

    param_A <- paste0("Condition", cond_A)
    param_B <- paste0("Condition", cond_B)

    if (!(param_A %in% valid_params && param_B %in% valid_params)) {
      message("  - Skipping comparison: ", comp_label, " (one or both conditions excluded or missing)")
      next
    }

    contrast_str <- paste0("Condition", cond_A, " - Condition", cond_B)

    tryCatch({
      contrast_mat <- limma::makeContrasts(contrasts = contrast_str, levels = design_mat)
      fit2 <- limma::contrasts.fit(fit, contrast_mat)
      fit2 <- limma::eBayes(fit2)

      res <- limma::topTable(fit2, number = Inf, adjust.method = adjust_method, sort.by = "none")
      res$Comparison <- comp_label

      if (data_type == "PTM" || data_type == "DPA" || data_type == "DPU") {
        res$Site_ID <- rownames(res)
        res$Protein <- rowData(se)$Protein
        res$Sequence <- rowData(se)$Sequence
      } else if (data_type == "Protein") {
        res$Protein <- rowData(se)$Protein
      }

      all_results[[comp_label]] <- res
    }, error = function(e) {
      message("Warning: Could not test comparison ", comp_label, ": ", e$message)
    })
  }

  if (length(all_results) > 0) {
    differential_results <- do.call(rbind, all_results)
    differential_results$Significant <- (abs(differential_results$logFC) >= lfc_threshold) & (differential_results$adj.P.Val <= alpha)

    if (data_type == "PTM") {
      write.table(differential_results, file = file.path(output_folder, "differential_phosphosites.txt"),
                  sep = "\t", row.names = FALSE, quote = FALSE)
      message("Saved differential phosphosite results")
    } else if (data_type == "DPA") {
      write.table(differential_results, file = file.path(output_folder, "dpa_results.txt"),
                  sep = "\t", row.names = FALSE, quote = FALSE)
      message("Saved DPA results")
    } else if (data_type == "DPU") {
      write.table(differential_results, file = file.path(output_folder, "dpu_results.txt"),
                  sep = "\t", row.names = FALSE, quote = FALSE)
      message("Saved DPU results")
    } else if (data_type == "Protein") {
      write.table(differential_results, file = file.path(output_folder, "differential_proteins.txt"),
                  sep = "\t", row.names = FALSE, quote = FALSE)
      message("Saved differential protein results")
    }

    return(list(results = all_results, comparisons = comparisons))
  }

  return(list(results = list(), comparisons = comparisons))
}

save_intensity_matrix <- function(se, output_folder, data_type = "PTM") {
  if (data_type == "PTM") {
    intensities <- as.data.frame(assay(se, "log2intensity"))
    intensities$Site_ID <- rowData(se)$Site_ID
    intensities$Protein <- rowData(se)$Protein
    intensities$Sequence <- rowData(se)$Sequence

    write.table(intensities, file = file.path(output_folder, "phosphosite_intensities.txt"),
                sep = "\t", row.names = FALSE, quote = FALSE)
  }
}
