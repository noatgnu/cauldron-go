perform_model_fitting <- function(pe, peptidoformAssayName, proteinAssayName, protein_col,
                                  analysis_type, model_run_effect, ridge_penalty,
                                  robust_regression, max_iterations) {
  message("[7/8] Fitting statistical models...")
  use_run_effect <- FALSE
  if (model_run_effect && "run" %in% colnames(colData(pe))) {
    n_runs <- length(unique(colData(pe)$run))
    if (n_runs > 1) {
      use_run_effect <- TRUE
      formula <- ~ 0 + condition + (1|run)
      message("  - Using formula with run effect (", n_runs, " runs): ", deparse(formula))
    } else {
      message("  - Warning: model_run_effect=TRUE but only 1 run detected. Using fixed effects only.")
      formula <- ~ 0 + condition
    }
  } else {
    formula <- ~ 0 + condition
  }

  tryCatch({
    pe <- msqrob(object = pe, i = proteinAssayName, formula = formula, ridge = ridge_penalty, robust = robust_regression, maxitRob = max_iterations)
  }, error = function(e) {
    stop("Protein model fitting failed with error: ", e$message)
  })

  if (!"msqrobModels" %in% colnames(rowData(pe[[proteinAssayName]]))) {
    stop("Protein model fitting failed. Check if there are enough observations per condition.")
  }

  protein_models <- rowData(pe[[proteinAssayName]])$msqrobModels
  n_null_models <- sum(sapply(protein_models, is.null))
  if (n_null_models > 0) {
    message("  - Warning: ", n_null_models, " proteins failed to fit. They will be excluded from testing.")
  }

  tryCatch({
    pe <- msqrob(object = pe, i = peptidoformAssayName, formula = formula, ridge = ridge_penalty, robust = robust_regression, maxitRob = max_iterations)
  }, error = function(e) {
    stop("PTM model fitting failed with error: ", e$message)
  })

  if (!"msqrobModels" %in% colnames(rowData(pe[[peptidoformAssayName]]))) {
    stop("PTM model fitting failed. Check if there are enough observations per condition.")
  }

  ptm_models <- rowData(pe[[peptidoformAssayName]])$msqrobModels
  n_null_models_ptm <- sum(sapply(ptm_models, is.null))
  if (n_null_models_ptm > 0) {
    message("  - Warning: ", n_null_models_ptm, " PTM features failed to fit. They will be excluded from testing.")
  }

  if (analysis_type %in% c("DPU", "both")) {
    ptm_prot_ids <- rowData(pe[[peptidoformAssayName]])[[protein_col]]

    ptm_mat <- assay(pe[[peptidoformAssayName]])
    prot_mat <- assay(pe[[proteinAssayName]])

    m <- match(ptm_prot_ids, rownames(pe[[proteinAssayName]]))
    valid_dpu <- !is.na(m)

    if (sum(valid_dpu) > 0) {
      dpu_mat <- ptm_mat[valid_dpu, ] - prot_mat[m[valid_dpu], ]

      ptm_rowdata <- rowData(pe[[peptidoformAssayName]])[valid_dpu, ]
      if ("msqrobModels" %in% colnames(ptm_rowdata)) {
        ptm_rowdata <- ptm_rowdata[, colnames(ptm_rowdata) != "msqrobModels", drop = FALSE]
      }
      dpu_se <- SummarizedExperiment(assays=list(dpu=dpu_mat), rowData=ptm_rowdata)
      colData(dpu_se) <- colData(pe)
      pe <- addAssay(pe, dpu_se, name = "peptideDPU")

      pe <- msqrob(object = pe, i = "peptideDPU", formula = formula, ridge = ridge_penalty, robust = robust_regression, maxitRob = max_iterations)
    }
  }

  return(pe)
}

perform_hypothesis_testing <- function(pe, peptidoformAssayName, proteinAssayName,
                                       comparison_file, analysis_type, alpha, lfc_threshold,
                                       adjust_method, output_folder) {
  message("[7/8] Hypothesis testing...")

  if (!is.null(comparison_file) && file.exists(comparison_file)) {
    comp_sep <- detect_delimiter(comparison_file)
    comparisons <- read.table(comparison_file, sep = comp_sep, header = TRUE,
                             stringsAsFactors = FALSE, check.names = FALSE)
    message("  - Loaded ", nrow(comparisons), " comparisons from file")
  } else {
    message("  - Performing all pairwise comparisons")
    conditions_levels <- levels(colData(pe)$condition)
    message("  - Available conditions: ", paste(conditions_levels, collapse = ", "))

    if (length(conditions_levels) < 2) {
      stop("Need at least 2 conditions for comparisons. Found: ", length(conditions_levels))
    }

    comparisons <- data.frame(
      comparison_label = character(),
      condition_A = character(),
      condition_B = character(),
      stringsAsFactors = FALSE
    )

    for (i in 1:(length(conditions_levels) - 1)) {
      for (j in (i + 1):length(conditions_levels)) {
        comparisons <- rbind(comparisons, data.frame(
          comparison_label = paste0(conditions_levels[j], "_vs_", conditions_levels[i]),
          condition_A = conditions_levels[i],
          condition_B = conditions_levels[j],
          stringsAsFactors = FALSE
        ))
      }
    }
    message("  - Generated ", nrow(comparisons), " pairwise comparisons")
  }

  if (nrow(comparisons) == 0) {
    stop("No comparisons to test")
  }

  all_dpa <- list()
  all_dpu <- list()
  all_protein <- list()
  valid_params <- paste0("condition", levels(colData(pe)$condition))

  if (analysis_type %in% c("DPA", "both")) {
    ptm_models <- rowData(pe[[peptidoformAssayName]])$msqrobModels
    valid_ptm <- !sapply(ptm_models, is.null)
    if (sum(valid_ptm) == 0) {
      stop("No valid PTM models for hypothesis testing. All features failed to fit.")
    }
    if (sum(!valid_ptm) > 0) {
      message("  - Filtering ", sum(!valid_ptm), " PTM features with failed models")
      ptm_assay_filtered <- pe[[peptidoformAssayName]][valid_ptm, ]
      pe <- removeAssay(pe, peptidoformAssayName)
      pe <- addAssay(pe, ptm_assay_filtered, name = peptidoformAssayName)
    }
  }

  protein_models_valid <- rowData(pe[[proteinAssayName]])$msqrobModels
  valid_proteins <- !sapply(protein_models_valid, is.null)
  if (sum(valid_proteins) == 0) {
    stop("No valid protein models for hypothesis testing. All features failed to fit.")
  }
  if (sum(!valid_proteins) > 0) {
    message("  - Filtering ", sum(!valid_proteins), " proteins with failed models")
    protein_assay_filtered <- pe[[proteinAssayName]][valid_proteins, ]
    pe_protein_filtered <- pe
    pe_protein_filtered <- removeAssay(pe_protein_filtered, proteinAssayName)
    pe_protein_filtered <- addAssay(pe_protein_filtered, protein_assay_filtered, name = proteinAssayName)
  } else {
    pe_protein_filtered <- pe
  }

  for (i in 1:nrow(comparisons)) {
    comp_label <- comparisons$comparison_label[i]
    cond_a <- make.names(comparisons$condition_A[i])
    cond_b <- make.names(comparisons$condition_B[i])

    param_a <- paste0("condition", cond_a)
    param_b <- paste0("condition", cond_b)

    if (!(param_a %in% valid_params && param_b %in% valid_params)) {
      message("  - Skipping comparison: ", comp_label, " (one or both conditions excluded or missing)")
      next
    }

    message("  - Testing: ", comp_label, " (", cond_b, " vs ", cond_a, ")")
    contrast_str <- paste0(param_b, " - ", param_a, " = 0")
    L <- makeContrast(contrast_str, parameterNames = valid_params)

    if (analysis_type %in% c("DPA", "both")) {
      result <- hypothesisTest(object = pe, i = peptidoformAssayName, contrast = L)
      result_df <- as.data.frame(rowData(result[[peptidoformAssayName]])[[colnames(L)]])
      if (adjust_method != "BH") { result_df$adjPval <- p.adjust(result_df$pval, method = adjust_method) }
      result_df$comparison <- comp_label
      result_df$condition_A <- comparisons$condition_A[i]
      result_df$condition_B <- comparisons$condition_B[i]
      result_df$feature <- rownames(result_df)
      result_df$significant <- (result_df$adjPval < alpha) & (abs(result_df$logFC) >= lfc_threshold)
      all_dpa[[comp_label]] <- result_df
    }

    if (analysis_type %in% c("DPU", "both") && "peptideDPU" %in% names(pe)) {
      result <- hypothesisTest(object = pe, i = "peptideDPU", contrast = L)
      result_df <- as.data.frame(rowData(result[["peptideDPU"]])[[colnames(L)]])
      if (adjust_method != "BH") { result_df$adjPval <- p.adjust(result_df$pval, method = adjust_method) }
      result_df$comparison <- comp_label
      result_df$condition_A <- comparisons$condition_A[i]
      result_df$condition_B <- comparisons$condition_B[i]
      result_df$feature <- rownames(result_df)
      result_df$significant <- (result_df$adjPval < alpha) & (abs(result_df$logFC) >= lfc_threshold)
      all_dpu[[comp_label]] <- result_df
    }

    result <- hypothesisTest(object = pe_protein_filtered, i = proteinAssayName, contrast = L)
    result_df <- as.data.frame(rowData(result[[proteinAssayName]])[[colnames(L)]])
    if (adjust_method != "BH") { result_df$adjPval <- p.adjust(result_df$pval, method = adjust_method) }
    result_df$comparison <- comp_label
    result_df$condition_A <- comparisons$condition_A[i]
    result_df$condition_B <- comparisons$condition_B[i]
    result_df$protein <- rownames(result_df)
    result_df$significant <- (result_df$adjPval < alpha) & (abs(result_df$logFC) >= lfc_threshold)
    all_protein[[comp_label]] <- result_df
  }

  if (length(all_dpa) > 0) {
    dpa_final <- do.call(rbind, all_dpa)
    dpa_final <- dpa_final[order(dpa_final$pval), ]
    write.table(dpa_final, file.path(output_folder, "dpa_results.txt"), sep="\t", quote=FALSE, row.names=FALSE)
    sig_count <- sum(dpa_final$adjPval < alpha, na.rm = TRUE)
    message("  - DPA: Found ", sig_count, " significant PTM features (adj. p-value < ", alpha, ")")
  }

  if (length(all_dpu) > 0) {
    dpu_final <- do.call(rbind, all_dpu)
    dpu_final <- dpu_final[order(dpu_final$pval), ]
    write.table(dpu_final, file.path(output_folder, "dpu_results.txt"), sep="\t", quote=FALSE, row.names=FALSE)
    sig_count <- sum(dpu_final$adjPval < alpha, na.rm = TRUE)
    message("  - DPU: Found ", sig_count, " significant PTM features (adj. p-value < ", alpha, ")")
  }

  if (length(all_protein) > 0) {
    protein_final <- do.call(rbind, all_protein)
    protein_final <- protein_final[order(protein_final$pval), ]
    write.table(protein_final, file.path(output_folder, "protein_results.txt"), sep="\t", quote=FALSE, row.names=FALSE)
    sig_count <- sum(protein_final$adjPval < alpha, na.rm = TRUE)
    message("  - Protein: Found ", sig_count, " significant proteins (adj. p-value < ", alpha, ")")
  }

  return(list(all_dpa = all_dpa, all_dpu = all_dpu, all_protein = all_protein))
}
