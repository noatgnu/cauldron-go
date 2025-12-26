library(MSstatsPTM)
library(ggplot2)
library(reshape2)

args <- commandArgs(trailingOnly = TRUE)

parse_args <- function(args) {
  parsed <- list()
  i <- 1
  while (i <= length(args)) {
    arg <- args[i]
    if (startsWith(arg, "--")) {
      key <- substring(arg, 3)
      if (i < length(args) && !startsWith(args[i + 1], "--")) {
        value <- args[i + 1]
        parsed[[key]] <- value
        i <- i + 2
      } else {
        parsed[[key]] <- TRUE
        i <- i + 1
      }
    } else {
      i <- i + 1
    }
  }
  return(parsed)
}

write_table_safe <- function(obj, name, filename) {
  if (is.null(obj)) {
    warning(paste(name, "is NULL; skipping write"))
    return(invisible(NULL))
  }
  fname <- file.path(output_folder, filename)
  tryCatch({
    write.table(obj, file = fname, sep = "\t", row.names = FALSE, quote = FALSE)
  }, error = function(e) {
    warning(paste("Failed to write", fname, ":", e$message))
  })
}

load_file_from_path <- function(path) {
    loaded_file <- NULL
    if (!is.null(path) && path != "") {
        if (grepl("\\.csv$", path)) {
                  loaded_file <- read.csv(path, header = TRUE, stringsAsFactors = FALSE)
                } else if (grepl("\\.tsv$|\\.txt$", path)) {
                  loaded_file <- read.delim(path, header = TRUE, sep = "\t", stringsAsFactors = FALSE)
                } else {
                  stop(paste("Unsupported file format:", path))
                }
            }
    loaded_file
}

params <- parse_args(args)
evidence_file <- params$evidence_file
annotation_file <- params$annotation_file
evidence_protein_file <- params$evidence_protein_file
annotation_protein_file <- params$annotation_protein_file
protein_groups_file <- params$protein_groups_file
sites_data_file <- params$sites_data_file
output_folder <- params$output_folder
fasta_file <- ifelse(is.null(params$fasta_file) || params$fasta_file == "", NULL, params$fasta_file)
fasta_protein_name <- ifelse(is.null(params$fasta_protein_name), "uniprot_ac", params$fasta_protein_name)

mod_id <- ifelse(is.null(params$mod_id), "\\(Phospho \\(STY\\)\\)", params$mod_id)
mod_num <- ifelse(is.null(params$mod_num), "Single", params$mod_num)
TMT_keyword <- ifelse(is.null(params$TMT_keyword), "Reporter intensity corrected", params$TMT_keyword)
ptm_keyword <- ifelse(is.null(params$ptm_keyword), "phos", params$ptm_keyword)
which_proteinid_ptm <- ifelse(is.null(params$which_proteinid_ptm), "Proteins", params$which_proteinid_ptm)
which_proteinid_protein <- ifelse(is.null(params$which_proteinid_protein), "Proteins", params$which_proteinid_protein)

remove_other_mods <- ifelse(is.null(params$remove_other_mods) || params$remove_other_mods == "true", TRUE, FALSE)
removeMpeptides <- ifelse(is.null(params$removeMpeptides) || params$removeMpeptides == "false", FALSE, TRUE)
removeOxidationMpeptides <- ifelse(is.null(params$removeOxidationMpeptides) || params$removeOxidationMpeptides == "false", FALSE, TRUE)
removeProtein_with1Peptide <- ifelse(is.null(params$removeProtein_with1Peptide) || params$removeProtein_with1Peptide == "false", FALSE, TRUE)
use_unmod_peptides <- ifelse(is.null(params$use_unmod_peptides) || params$use_unmod_peptides == "true", TRUE, FALSE)

summarization_method <- ifelse(is.null(params$summarization_method), "msstats", params$summarization_method)
global_norm <- ifelse(is.null(params$global_norm) || params$global_norm == "true", TRUE, FALSE)
global_norm_PTM <- ifelse(is.null(params$global_norm_PTM) || params$global_norm_PTM == "true", TRUE, FALSE)
reference_norm <- ifelse(is.null(params$reference_norm) || params$reference_norm == "true", TRUE, FALSE)
reference_norm_PTM <- ifelse(is.null(params$reference_norm_PTM) || params$reference_norm_PTM == "true", TRUE, FALSE)
remove_norm_channel <- ifelse(is.null(params$remove_norm_channel) || params$remove_norm_channel == "true", TRUE, FALSE)
remove_empty_channel <- ifelse(is.null(params$remove_empty_channel) || params$remove_empty_channel == "true", TRUE, FALSE)
MBimpute <- ifelse(is.null(params$MBimpute) || params$MBimpute == "true", TRUE, FALSE)
MBimpute_PTM <- ifelse(is.null(params$MBimpute_PTM) || params$MBimpute_PTM == "true", TRUE, FALSE)
maxQuantileforCensored <- ifelse(is.null(params$maxQuantileforCensored), NULL, as.numeric(params$maxQuantileforCensored))

comparison_matrix_file <- params$comparison_matrix_file
moderated <- ifelse(is.null(params$moderated) || params$moderated == "false", FALSE, TRUE)
adj_method <- ifelse(is.null(params$adj_method), "BH", params$adj_method)
log_base <- ifelse(is.null(params$log_base), 2, as.numeric(params$log_base))
save_fitted_models <- ifelse(is.null(params$save_fitted_models) || params$save_fitted_models == "false", FALSE, TRUE)

evidence_ptm <- load_file_from_path(evidence_file)
annotation_ptm <- load_file_from_path(annotation_file)
evidence_prot <- load_file_from_path(evidence_protein_file)
annotation_prot <- load_file_from_path(annotation_protein_file)
protein_groups <- load_file_from_path(protein_groups_file)
sites_data <- load_file_from_path(sites_data_file)

cat("Checking annotation file before conversion...\n")
if (!is.null(annotation_ptm)) {
    cat(paste("  Annotation columns:", paste(names(annotation_ptm), collapse=", "), "\n"))
    cat(paste("  Annotation rows:", nrow(annotation_ptm), "\n"))

    if ("Channel" %in% names(annotation_ptm)) {
        cat(paste("  Annotation unique Channels:", paste(unique(annotation_ptm$Channel), collapse=", "), "\n"))
    }
    if ("Run" %in% names(annotation_ptm)) {
        cat(paste("  Annotation unique Runs:", paste(unique(annotation_ptm$Run), collapse=", "), "\n"))
    }
}

cat("Converting MaxQuant data to MSstatsPTM format...\n")
data <- MaxQtoMSstatsPTMFormat(
        evidence = evidence_ptm,
        annotation = annotation_ptm,
        fasta_path = fasta_file,
        fasta_protein_name = fasta_protein_name,
        mod_id = mod_id,
        evidence_prot = evidence_prot,
        proteinGroups = protein_groups,
        annotation_protein = annotation_prot,
        use_unmod_peptides = use_unmod_peptides,
        labeling_type = "TMT",
        mod_num = mod_num,
        ptm_keyword = ptm_keyword,
        which_proteinid_ptm = which_proteinid_ptm,
        which_proteinid_protein = which_proteinid_protein,
        remove_other_mods = remove_other_mods,
        removeMpeptides = removeMpeptides,
        removeOxidationMpeptides = removeOxidationMpeptides,
        removeProtein_with1Peptide = removeProtein_with1Peptide,
        use_log_file = FALSE,
        append = FALSE,
        verbose = TRUE,
        log_file_path = NULL
        )

cat("Removing Norm from annotation after conversion for contrast matrix creation...\n")
if (!is.null(annotation_ptm) && "Condition" %in% names(annotation_ptm)) {
    before_rows <- nrow(annotation_ptm)
    annotation_ptm <- annotation_ptm[annotation_ptm$Condition != "Norm", ]
    cat(paste("  PTM annotation rows:", before_rows, "->", nrow(annotation_ptm), "(removed Norm)\n"))
}

if (!is.null(annotation_prot) && "Condition" %in% names(annotation_prot)) {
    before_rows <- nrow(annotation_prot)
    annotation_prot <- annotation_prot[annotation_prot$Condition != "Norm", ]
    cat(paste("  Protein annotation rows:", before_rows, "->", nrow(annotation_prot), "(removed Norm)\n"))
}

cat("Checking converted data structure...\n")
cat(paste("Data is null:", is.null(data), "\n"))
cat(paste("Data length:", length(data), "\n"))
if (!is.null(data) && length(data) > 0) {
    cat(paste("PTM data columns:", paste(names(data$PTM), collapse=", "), "\n"))
    cat(paste("PTM data rows:", nrow(data$PTM), "\n"))

    if ("Channel" %in% names(data$PTM)) {
        unique_channels <- unique(data$PTM$Channel)
        cat(paste("  PTM unique Channels after conversion:", paste(head(unique_channels, 20), collapse=", "), "\n"))
        na_channel_count <- sum(is.na(data$PTM$Channel))
        cat(paste("  PTM rows with NA Channel:", na_channel_count, "\n"))
    }

    if (!is.null(data$PROTEIN)) {
        cat(paste("PROTEIN data columns:", paste(names(data$PROTEIN), collapse=", "), "\n"))
        cat(paste("PROTEIN data rows:", nrow(data$PROTEIN), "\n"))
    }
}

cat("Checking data before summarization...\n")
if (!is.null(data$PTM)) {
    if ("Run" %in% names(data$PTM)) {
        ptm_runs <- unique(data$PTM$Run)
        cat(paste("  PTM unique Runs:", paste(ptm_runs, collapse=", "), "\n"))
    }
    if ("Condition" %in% names(data$PTM)) {
        ptm_conds <- unique(data$PTM$Condition)
        cat(paste("  PTM unique Conditions:", paste(ptm_conds, collapse=", "), "\n"))

        na_count <- sum(is.na(data$PTM$Condition))
        if (na_count > 0) {
            cat(paste("  WARNING: Found", na_count, "rows with NA Condition\n"))
            if ("Channel" %in% names(data$PTM)) {
                na_channels <- unique(data$PTM$Channel[is.na(data$PTM$Condition)])
                cat(paste("  NA Condition channels:", paste(na_channels, collapse=", "), "\n"))
            }
            if ("Raw.file" %in% names(data$PTM)) {
                na_raw <- unique(data$PTM$Raw.file[is.na(data$PTM$Condition)])
                cat(paste("  NA Condition raw files:", paste(head(na_raw, 5), collapse=", "), "\n"))
            }
        }
    }
    if ("Mixture" %in% names(data$PTM)) {
        ptm_mix <- unique(data$PTM$Mixture)
        cat(paste("  PTM unique Mixtures:", paste(ptm_mix, collapse=", "), "\n"))
    }

    cat("Removing rows with NA Condition or Mixture...\n")
    before_count <- nrow(data$PTM)
    data$PTM <- data$PTM[!is.na(data$PTM$Condition) & !is.na(data$PTM$Mixture), ]
    after_count <- nrow(data$PTM)
    cat(paste("  Removed", before_count - after_count, "rows with NA values\n"))
    cat(paste("  Remaining rows:", after_count, "\n"))

    if (!is.null(data$PROTEIN)) {
        before_count_prot <- nrow(data$PROTEIN)
        data$PROTEIN <- data$PROTEIN[!is.na(data$PROTEIN$Condition) & !is.na(data$PROTEIN$Mixture), ]
        after_count_prot <- nrow(data$PROTEIN)
        cat(paste("  PROTEIN: Removed", before_count_prot - after_count_prot, "rows with NA values\n"))
    }
}

cat("Summarizing TMT data...\n")

summarized <- dataSummarizationPTM_TMT(
              data,
              method = summarization_method,
              global_norm = global_norm,
              global_norm.PTM = global_norm_PTM,
              reference_norm = reference_norm,
              reference_norm.PTM = reference_norm_PTM,
              remove_norm_channel = remove_norm_channel,
              remove_empty_channel = remove_empty_channel,
              MBimpute = MBimpute,
              MBimpute.PTM = MBimpute_PTM,
              maxQuantileforCensored = maxQuantileforCensored,
              use_log_file = FALSE,
              append = FALSE,
              verbose = TRUE,
              log_file_path = NULL
              )

cat("Checking summarized data structure...\n")
cat(paste("Summarized is null:", is.null(summarized), "\n"))
if (!is.null(summarized)) {
    cat(paste("Summarized length:", length(summarized), "\n"))
    cat(paste("Summarized names:", paste(names(summarized), collapse=", "), "\n"))
    if (!is.null(summarized$PTM)) {
        cat(paste("Summarized PTM columns:", paste(names(summarized$PTM), collapse=", "), "\n"))
        cat(paste("Summarized PTM rows:", nrow(summarized$PTM), "\n"))
    }
    if (!is.null(summarized$PROTEIN)) {
        cat(paste("Summarized PROTEIN columns:", paste(names(summarized$PROTEIN), collapse=", "), "\n"))
        cat(paste("Summarized PROTEIN rows:", nrow(summarized$PROTEIN), "\n"))
    }
}

cat("Creating wide-format intensity tables...\n")
create_wide_intensity_table <- function(data, id_col, intensity_col) {
    if (is.null(data)) return(NULL)

    library(tidyr)
    library(dplyr)

    data_df <- as.data.frame(data)

    for (col in names(data_df)) {
        if (is.list(data_df[[col]])) {
            data_df[[col]] <- as.character(data_df[[col]])
        }
    }

    cat(paste("  Available columns:", paste(names(data_df), collapse=", "), "\n"))

    group_col <- if ("GROUP" %in% names(data_df)) "GROUP" else "Condition"
    subject_col <- if ("SUBJECT" %in% names(data_df)) "SUBJECT" else "BioReplicate"
    actual_intensity_col <- if (intensity_col %in% names(data_df)) intensity_col else if ("Abundance" %in% names(data_df)) "Abundance" else "LogIntensities"

    cat(paste("  Using columns - ID:", id_col, ", Group:", group_col, ", Subject:", subject_col, ", Intensity:", actual_intensity_col, "\n"))

    required_cols <- c(id_col, group_col, subject_col, actual_intensity_col)
    missing_cols <- setdiff(required_cols, names(data_df))
    if (length(missing_cols) > 0) {
        stop(paste("Missing required columns:", paste(missing_cols, collapse=", ")))
    }

    wide_data <- data_df %>%
        select(all_of(c(id_col, group_col, subject_col, actual_intensity_col))) %>%
        mutate(sample_id = paste(!!sym(group_col), !!sym(subject_col), sep = ".")) %>%
        select(all_of(c(id_col, "sample_id", actual_intensity_col))) %>%
        distinct() %>%
        pivot_wider(
            id_cols = all_of(id_col),
            names_from = sample_id,
            values_from = all_of(actual_intensity_col),
            values_fn = mean
        )

    return(wide_data)
}

peptide_intensity_wide <- NULL
protein_intensity_wide <- NULL

if (!is.null(summarized$PTM) && !is.null(summarized$PTM$ProteinLevelData)) {
    tryCatch({
        peptide_intensity_wide <- create_wide_intensity_table(summarized$PTM$ProteinLevelData, "Protein", "LogIntensities")
        if (!is.null(peptide_intensity_wide)) {
            cat(paste("Created peptide intensity table:", nrow(peptide_intensity_wide), "peptides x", ncol(peptide_intensity_wide) - 1, "samples\n"))
        }
    }, error = function(e) {
        warning(paste("Failed to create peptide intensity table:", e$message))
    })
}

if (!is.null(summarized$PROTEIN) && !is.null(summarized$PROTEIN$ProteinLevelData)) {
    tryCatch({
        protein_intensity_wide <- create_wide_intensity_table(summarized$PROTEIN$ProteinLevelData, "Protein", "LogIntensities")
        if (!is.null(protein_intensity_wide)) {
            cat(paste("Created protein intensity table:", nrow(protein_intensity_wide), "proteins x", ncol(protein_intensity_wide) - 1, "samples\n"))
        }
    }, error = function(e) {
        warning(paste("Failed to create protein intensity table:", e$message))
    })
}

if (is.null(output_folder) || output_folder == "") output_folder <- "."
dir.create(output_folder, recursive = TRUE, showWarnings = FALSE)

if (!is.null(peptide_intensity_wide)) {
    fname <- file.path(output_folder, "peptidoform_intensities.txt")
    tryCatch({
        write.table(peptide_intensity_wide, file = fname, sep = "\t", row.names = FALSE, quote = FALSE)
        cat(paste("Saved peptidoform intensity table to:", fname, "\n"))
    }, error = function(e) {
        warning(paste("Failed to write peptidoform intensity table:", e$message))
    })
}

if (!is.null(protein_intensity_wide)) {
    fname <- file.path(output_folder, "protein_intensities.txt")
    tryCatch({
        write.table(protein_intensity_wide, file = fname, sep = "\t", row.names = FALSE, quote = FALSE)
        cat(paste("Saved protein intensity table to:", fname, "\n"))
    }, error = function(e) {
        warning(paste("Failed to write protein intensity table:", e$message))
    })
}

if (is.null(comparison_matrix_file) || comparison_matrix_file == "") {
    conditions <- unique(annotation_ptm$Condition)
    conditions <- sort(conditions)
    cat(paste("Creating pairwise comparison matrix from conditions:", paste(conditions, collapse=", "), "\n"))

    comparisons <- list()
    for (i in 1:(length(conditions)-1)) {
        for (j in (i+1):length(conditions)) {
            comp_name <- paste(conditions[i], conditions[j], sep="-")
            comp_row <- rep(0, length(conditions))
            comp_row[i] <- 1
            comp_row[j] <- -1
            comparisons[[comp_name]] <- comp_row
        }
    }

    comparison_matrix <- do.call(rbind, comparisons)
    colnames(comparison_matrix) <- conditions
    cat("Generated pairwise comparison matrix:\n")
    print(comparison_matrix)
} else {
    if (grepl("\\.csv$", comparison_matrix_file)) {
          comparison_matrix <- read.csv(comparison_matrix_file, header = TRUE, stringsAsFactors = FALSE)
        } else if (grepl("\\.tsv$|\\.txt$", comparison_matrix_file)) {
          comparison_matrix <- read.delim(comparison_matrix_file, header = TRUE, sep = "\t", stringsAsFactors = FALSE)
        } else {
          stop(paste("Unsupported file format:", comparison_matrix_file))
        }
    cat("Loaded custom comparison matrix\n")
}

cat("Performing group comparison...\n")
group_comparison <- groupComparisonPTM(
    summarized,
    ptm_label_type = "TMT",
    protein_label_type = "TMT",
    contrast.matrix = comparison_matrix,
    moderated = moderated,
    adj.method = adj_method,
    save_fitted_models = save_fitted_models,
    log_base = log_base,
    use_log_file = FALSE,
    append = FALSE,
    verbose = TRUE,
    log_file_path = NULL
)

write_table_safe(group_comparison$PTM.Model, "PTM", "group_comparison_ptm.txt")
write_table_safe(group_comparison$PROTEIN.Model, "Protein", "group_comparison_protein.txt")
write_table_safe(group_comparison$ADJUSTED.Model, "Adjusted", "group_comparison_adjusted.txt")

plot_boxplot_ggplot <- function(data_long, title) {
  p <- ggplot(data_long, aes(x = Sample, y = Intensity, fill = Condition)) +
    geom_boxplot(outlier.size = 0.5) + theme_minimal() +
    labs(title = title, x = "", y = "Log2 Intensity") +
    theme(axis.text.x = element_text(angle = 45, hjust = 1), legend.position = "bottom")
  return(p)
}

plot_volcano <- function(df, title, comparison) {
  df$significant <- ifelse(abs(df$log2FC) > 1 & df$adj.pvalue < 0.05, "Significant", "Not Significant")
  p <- ggplot(df, aes(x = log2FC, y = -log10(adj.pvalue), color = significant)) +
    geom_point(alpha = 0.6, size = 1.5) +
    scale_color_manual(values = c("Significant" = "red", "Not Significant" = "gray")) +
    labs(title = paste(title, "-", comparison), x = "Log2 Fold Change", y = "-Log10 Adjusted P-value") +
    theme_minimal() + geom_hline(yintercept = -log10(0.05), linetype = "dashed", color = "blue") +
    geom_vline(xintercept = c(-1, 1), linetype = "dashed", color = "blue")
  return(p)
}

cat("Generating quality control and volcano plots...\n")

if (!is.null(protein_quant_msstats) && nrow(protein_quant_msstats) > 0) {
  pdf(file.path(output_folder, "protein_qc_plots.pdf"), width = 14, height = 10)

  protein_long <- protein_quant_msstats[, c("Protein", "Run", "Channel", "Condition", "Abundance")]
  protein_long$Sample <- paste(protein_long$Run, protein_long$Channel, sep = "_")

  if (nrow(protein_long) > 0) {
    p <- plot_boxplot_ggplot(protein_long, "Protein Intensities")
    print(p)

    protein_wide_pca <- dcast(protein_long, Protein ~ Sample, value.var = "Abundance")
    rownames(protein_wide_pca) <- protein_wide_pca$Protein
    protein_wide_pca$Protein <- NULL
    protein_wide_pca <- protein_wide_pca[complete.cases(protein_wide_pca), ]

    if (nrow(protein_wide_pca) > 2 && ncol(protein_wide_pca) > 2) {
      pca_result <- prcomp(t(protein_wide_pca), scale. = TRUE)
      df_pca <- data.frame(
        Sample = rownames(pca_result$x),
        PC1 = pca_result$x[, 1],
        PC2 = pca_result$x[, 2]
      )

      df_pca$Condition <- sapply(df_pca$Sample, function(s) {
        idx <- which(paste(protein_long$Run, protein_long$Channel, sep = "_") == s)[1]
        if (length(idx) > 0) protein_long$Condition[idx] else NA
      })

      var_explained <- summary(pca_result)$importance[2, 1:2] * 100
      p_pca <- ggplot(df_pca, aes(x = PC1, y = PC2, color = Condition, label = Sample)) +
        geom_point(size = 3) +
        labs(title = "PCA - Protein Level",
             x = sprintf("PC1 (%.1f%%)", var_explained[1]),
             y = sprintf("PC2 (%.1f%%)", var_explained[2])) +
        theme_minimal()
      print(p_pca)
    }
  }

  dev.off()
}

if (!is.null(group_comparison$PROTEIN.Model)) {
  pdf(file.path(output_folder, "protein_volcano_plots.pdf"), width = 14, height = 10)
  comparisons <- unique(group_comparison$PROTEIN.Model$Label)
  for (comp in comparisons) {
    comp_data <- group_comparison$PROTEIN.Model[group_comparison$PROTEIN.Model$Label == comp, ]
    print(plot_volcano(comp_data, "Volcano Plot - Protein Level", comp))
  }
  dev.off()
}

if (!is.null(ptm_quant_msstats) && nrow(ptm_quant_msstats) > 0) {
  pdf(file.path(output_folder, "ptm_qc_plots.pdf"), width = 14, height = 10)

  ptm_long <- ptm_quant_msstats[, c("PTM_id", "Run", "Channel", "Condition", "Abundance")]
  ptm_long$Sample <- paste(ptm_long$Run, ptm_long$Channel, sep = "_")

  if (nrow(ptm_long) > 0) {
    p <- plot_boxplot_ggplot(ptm_long, "PTM/Peptidoform Intensities")
    print(p)

    ptm_wide_pca <- dcast(ptm_long, PTM_id ~ Sample, value.var = "Abundance")
    rownames(ptm_wide_pca) <- ptm_wide_pca$PTM_id
    ptm_wide_pca$PTM_id <- NULL
    ptm_wide_pca <- ptm_wide_pca[complete.cases(ptm_wide_pca), ]

    if (nrow(ptm_wide_pca) > 2 && ncol(ptm_wide_pca) > 2) {
      pca_result <- prcomp(t(ptm_wide_pca), scale. = TRUE)
      df_pca <- data.frame(
        Sample = rownames(pca_result$x),
        PC1 = pca_result$x[, 1],
        PC2 = pca_result$x[, 2]
      )

      df_pca$Condition <- sapply(df_pca$Sample, function(s) {
        idx <- which(paste(ptm_long$Run, ptm_long$Channel, sep = "_") == s)[1]
        if (length(idx) > 0) ptm_long$Condition[idx] else NA
      })

      var_explained <- summary(pca_result)$importance[2, 1:2] * 100
      p_pca <- ggplot(df_pca, aes(x = PC1, y = PC2, color = Condition, label = Sample)) +
        geom_point(size = 3) +
        labs(title = "PCA - PTM/Peptidoform Level",
             x = sprintf("PC1 (%.1f%%)", var_explained[1]),
             y = sprintf("PC2 (%.1f%%)", var_explained[2])) +
        theme_minimal()
      print(p_pca)
    }
  }

  dev.off()
}

if (!is.null(group_comparison$PTM.Model) && !is.null(group_comparison$ADJUSTED.Model)) {
  pdf(file.path(output_folder, "ptm_volcano_plots.pdf"), width = 14, height = 10)
  comparisons <- unique(group_comparison$PTM.Model$Label)
  for (comp in comparisons) {
    dpa_data <- group_comparison$PTM.Model[group_comparison$PTM.Model$Label == comp, ]
    print(plot_volcano(dpa_data, "Volcano Plot - DPA (PTM Abundance)", comp))

    dpu_data <- group_comparison$ADJUSTED.Model[group_comparison$ADJUSTED.Model$Label == comp, ]
    print(plot_volcano(dpu_data, "Volcano Plot - DPU (PTM Usage)", comp))
  }
  dev.off()
}

cat("MSstatsPTM TMT analysis completed successfully\n")
cat(paste("Results saved to:", output_folder, "\n"))
