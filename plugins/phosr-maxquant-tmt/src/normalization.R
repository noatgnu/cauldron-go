filter_and_normalize_se <- function(se, normalize_method, impute, impute_order = "after", col_filter = 0.7, row_filter = 0.7) {
  message("\n[5/9] Filtering, normalization, and imputation for PTM...")

  col_na_frac <- colMeans(is.na(assay(se, "log2intensity")))
  se <- se[, col_na_frac < col_filter]
  message("Retained ", ncol(se), " samples after column filtering (< ", col_filter * 100, "% missing)")

  row_na_frac <- rowMeans(is.na(assay(se, "log2intensity")))
  se <- se[row_na_frac < row_filter, ]
  message("Retained ", nrow(se), " sites after row filtering (< ", row_filter * 100, "% missing)")

  if (impute_order == "before" && impute != "none") {
    message("Imputing missing values with method: ", impute, " (before normalization)")
    mat <- assay(se, "log2intensity")
    mat_imputed <- MsCoreUtils::impute_matrix(mat, method = impute)
    assay(se, "log2intensity") <- mat_imputed
  }

  if (normalize_method != "none") {
    message("Normalizing with method: ", normalize_method)
    mat <- assay(se, "log2intensity")

    if (normalize_method == "center.median") {
      col_medians <- apply(mat, 2, median, na.rm = TRUE)
      global_median <- median(col_medians, na.rm = TRUE)
      mat <- sweep(mat, 2, col_medians - global_median, "-")
    } else if (normalize_method == "center.mean") {
      col_means <- colMeans(mat, na.rm = TRUE)
      global_mean <- mean(col_means, na.rm = TRUE)
      mat <- sweep(mat, 2, col_means - global_mean, "-")
    } else if (normalize_method == "quantiles") {
      mat <- limma::normalizeBetweenArrays(mat, method = "quantile")
    } else if (normalize_method == "quantiles.robust") {
      mat <- limma::normalizeBetweenArrays(mat, method = "Rquantile")
    }

    assay(se, "log2intensity") <- mat
  }

  if (impute_order == "after" && impute != "none") {
    message("Imputing missing values with method: ", impute, " (after normalization)")
    mat <- assay(se, "log2intensity")
    mat_imputed <- MsCoreUtils::impute_matrix(mat, method = impute)
    assay(se, "log2intensity") <- mat_imputed
  }

  return(se)
}

filter_and_normalize_protein <- function(se_protein, normalize_method, impute, impute_order = "after", output_folder, col_filter = 0.7, row_filter = 0.7) {
  col_na_frac_prot <- colMeans(is.na(assay(se_protein, "log2intensity")))
  se_protein <- se_protein[, col_na_frac_prot < col_filter]
  message("  - Retained ", ncol(se_protein), " samples after column filtering (< ", col_filter * 100, "% missing)")

  row_na_frac_prot <- rowMeans(is.na(assay(se_protein, "log2intensity")))
  se_protein <- se_protein[row_na_frac_prot < row_filter, ]
  message("  - Retained ", nrow(se_protein), " proteins after row filtering (< ", row_filter * 100, "% missing)")

  if (impute_order == "before" && impute != "none") {
    message("  - Imputing protein missing values with method: ", impute, " (before normalization)")
    mat_prot <- assay(se_protein, "log2intensity")
    mat_prot_imputed <- MsCoreUtils::impute_matrix(mat_prot, method = impute)
    assay(se_protein, "log2intensity") <- mat_prot_imputed
  }

  if (normalize_method != "none") {
    message("  - Normalizing proteins with method: ", normalize_method)
    mat_prot <- assay(se_protein, "log2intensity")

    if (normalize_method == "center.median") {
      col_medians <- apply(mat_prot, 2, median, na.rm = TRUE)
      global_median <- median(col_medians, na.rm = TRUE)
      mat_prot <- sweep(mat_prot, 2, col_medians - global_median, "-")
    } else if (normalize_method == "center.mean") {
      col_means <- colMeans(mat_prot, na.rm = TRUE)
      global_mean <- mean(col_means, na.rm = TRUE)
      mat_prot <- sweep(mat_prot, 2, col_means - global_mean, "-")
    } else if (normalize_method == "quantiles") {
      mat_prot <- limma::normalizeBetweenArrays(mat_prot, method = "quantile")
    } else if (normalize_method == "quantiles.robust") {
      mat_prot <- limma::normalizeBetweenArrays(mat_prot, method = "Rquantile")
    }

    assay(se_protein, "log2intensity") <- mat_prot
  }

  if (impute_order == "after" && impute != "none") {
    message("  - Imputing protein missing values with method: ", impute, " (after normalization)")
    mat_prot <- assay(se_protein, "log2intensity")
    mat_prot_imputed <- MsCoreUtils::impute_matrix(mat_prot, method = impute)
    assay(se_protein, "log2intensity") <- mat_prot_imputed
  }

  protein_intensities <- as.data.frame(assay(se_protein, "log2intensity"))
  protein_intensities$Protein <- rowData(se_protein)$Protein
  write.table(protein_intensities, file = file.path(output_folder, "protein_intensities.txt"),
              sep = "\t", row.names = FALSE, quote = FALSE)
  message("  - Saved protein intensities")

  return(se_protein)
}
