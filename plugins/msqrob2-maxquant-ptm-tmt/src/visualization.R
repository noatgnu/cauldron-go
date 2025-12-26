plot_boxplot_ggplot <- function(pe_obj, i, title) {
  mat <- assay(pe_obj[[i]])
  if (ncol(mat) == 0) return(NULL)
  df_long <- reshape2::melt(mat)
  colnames(df_long) <- c("Feature", "Sample", "Intensity")
  cd <- as.data.frame(colData(pe_obj))
  cd$Sample <- rownames(cd)
  df_long <- merge(df_long, cd, by = "Sample")
  p <- ggplot(df_long, aes(x = Sample, y = Intensity, fill = condition)) +
    geom_boxplot(outlier.size = 0.5) + theme_minimal() +
    theme(axis.text.x = element_text(angle = 90, vjust = 0.5, hjust=1)) +
    labs(title = title, y = "Log2 Intensity")
  return(p)
}

plot_volcano <- function(df, title, comparison, alpha, lfc_threshold) {
  if (is.null(df) || nrow(df) == 0) return(NULL)
  v_data <- df[!is.na(df$logFC) & !is.na(df$adjPval), ]
  if (nrow(v_data) == 0) return(NULL)
  p <- ggplot(v_data, aes(x = logFC, y = -log10(adjPval), color = significant)) +
    geom_point(alpha = 0.5, size = 0.8) +
    scale_color_manual(values = c("gray", "red")) +
    theme_minimal() +
    geom_hline(yintercept = -log10(alpha), linetype = "dashed", color = "blue") +
    {if(lfc_threshold > 0) geom_vline(xintercept = c(-lfc_threshold, lfc_threshold), linetype = "dashed", color = "blue")} +
    labs(title = paste(title, ":", comparison))
  return(p)
}

generate_qc_plots <- function(pe, peptidoformAssayName, proteinAssayName, output_folder) {
  message("[8/8] Generating quality control and volcano plots...")

  pdf(file.path(output_folder, "protein_qc_plots.pdf"), width = 14, height = 10)
  if ("protein_raw" %in% names(pe)) {
    p <- plot_boxplot_ggplot(pe, "protein_raw", "Protein Intensities (Before Norm)")
    if (!is.null(p)) print(p)
  }
  p <- plot_boxplot_ggplot(pe, proteinAssayName, "Protein Intensities (Global Normalized)")
  if (!is.null(p)) print(p)

  protein_matrix <- assay(pe[[proteinAssayName]])
  pca_data_prot <- t(na.omit(protein_matrix))
  if (nrow(pca_data_prot) > 2 && ncol(pca_data_prot) > 2) {
    col_vars_prot <- apply(pca_data_prot, 2, var, na.rm = TRUE)
    pca_data_prot <- pca_data_prot[, col_vars_prot > 0 & !is.na(col_vars_prot), drop = FALSE]
    if (ncol(pca_data_prot) > 2) {
      pca_result_prot <- prcomp(pca_data_prot, scale. = TRUE)
      df_pca_prot <- as.data.frame(pca_result_prot$x)
      df_pca_prot$Sample <- rownames(df_pca_prot)
      cd <- as.data.frame(colData(pe))
      cd$Sample <- rownames(cd)
      df_pca_prot <- merge(df_pca_prot, cd, by = "Sample")
      p_pca_prot <- ggplot(df_pca_prot, aes(x = PC1, y = PC2, color = condition, label = Sample)) +
        geom_point(size = 3) + geom_text(vjust = 1.5, size = 3) + theme_minimal() +
        labs(title = "PCA - Protein Level")
      print(p_pca_prot)
    }
  }
  dev.off()

  pdf(file.path(output_folder, "ptm_qc_plots.pdf"), width = 14, height = 10)
  if ("peptidoform_raw" %in% names(pe)) {
    p <- plot_boxplot_ggplot(pe, "peptidoform_raw", "PTM/Peptidoform Intensities (Before Norm)")
    if (!is.null(p)) print(p)
  }
  p <- plot_boxplot_ggplot(pe, peptidoformAssayName, "PTM/Peptidoform Intensities (Normalized)")
  if (!is.null(p)) print(p)

  peptidoform_matrix <- assay(pe[[peptidoformAssayName]])
  pca_data <- t(na.omit(peptidoform_matrix))
  if (nrow(pca_data) > 2 && ncol(pca_data) > 2) {
    col_vars <- apply(pca_data, 2, var, na.rm = TRUE)
    pca_data <- pca_data[, col_vars > 0 & !is.na(col_vars), drop = FALSE]
    if (ncol(pca_data) > 2) {
      pca_result <- prcomp(pca_data, scale. = TRUE)
      df_pca <- as.data.frame(pca_result$x)
      df_pca$Sample <- rownames(df_pca)
      cd <- as.data.frame(colData(pe))
      cd$Sample <- rownames(cd)
      df_pca <- merge(df_pca, cd, by = "Sample")
      p_pca <- ggplot(df_pca, aes(x = PC1, y = PC2, color = condition, label = Sample)) +
        geom_point(size = 3) + geom_text(vjust = 1.5, size = 3) + theme_minimal() +
        labs(title = "PCA - PTM/Peptidoform Level")
      print(p_pca)
    }
  }
  dev.off()
}

generate_volcano_plots <- function(all_dpa, all_dpu, all_protein, alpha, lfc_threshold, output_folder) {
  pdf(file.path(output_folder, "protein_volcano_plots.pdf"), width = 14, height = 10)
  for (comp in names(all_protein)) {
    print(plot_volcano(all_protein[[comp]], "Volcano Plot - Protein Level", comp, alpha, lfc_threshold))
  }
  dev.off()

  pdf(file.path(output_folder, "ptm_volcano_plots.pdf"), width = 14, height = 10)
  comp_names <- names(all_protein)
  if (length(comp_names) > 0) {
    for (comp in comp_names) {
      if (exists("all_dpa") && comp %in% names(all_dpa)) print(plot_volcano(all_dpa[[comp]], "Volcano Plot - DPA (PTM Abundance)", comp, alpha, lfc_threshold))
      if (exists("all_dpu") && comp %in% names(all_dpu)) print(plot_volcano(all_dpu[[comp]], "Volcano Plot - DPU (PTM Usage)", comp, alpha, lfc_threshold))
    }
  }
  dev.off()
}
