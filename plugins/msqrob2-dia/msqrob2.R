library(QFeatures)
library(msqrob2)
library(limma)
library(ggplot2)
library(reshape2)

detect_delimiter <- function(filepath) {
  ext <- tolower(tools::file_ext(filepath))
  if (ext == "csv") {
    return(",")
  } else if (ext %in% c("tsv", "txt")) {
    return("\t")
  } else {
    return("\t")
  }
}

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

params <- parse_args(args)

input_file <- params$input_file
output_folder <- params$output_folder
annotation_file <- params$annotation_file
comparison_file <- params$comparison_file
feature_id_col <- params$feature_id_col
protein_col <- params$protein_col
log2_transform <- !is.null(params$log2_transform) && params$log2_transform != FALSE
filter_min_identified <- ifelse(is.null(params$filter_min_identified), 2, as.numeric(params$filter_min_identified))
col_filter <- ifelse(is.null(params$col_filter), 0.7, as.numeric(params$col_filter))
row_filter <- ifelse(is.null(params$row_filter), 0.7, as.numeric(params$row_filter))
impute_order <- ifelse(is.null(params$impute_order), "after", params$impute_order)
impute_method <- params$impute
normalize_method <- ifelse(is.null(params$normalize_method), "center.median", params$normalize_method)
aggregation_order <- ifelse(is.null(params$aggregation_order), "after", params$aggregation_order)
aggregation_method <- ifelse(is.null(params$aggregation_method), "robust", params$aggregation_method)
ridge_penalty <- ifelse(is.null(params$ridge_penalty), 0, as.numeric(params$ridge_penalty))
robust_regression <- !is.null(params$robust_regression) && params$robust_regression != FALSE
max_iterations <- ifelse(is.null(params$max_iterations), 20, as.numeric(params$max_iterations))
adjust_method <- ifelse(is.null(params$adjust_method), "BH", params$adjust_method)
alpha <- ifelse(is.null(params$alpha), 0.05, as.numeric(params$alpha))
lfc_threshold <- ifelse(is.null(params$lfc_threshold), 0, as.numeric(params$lfc_threshold))
exclude_conditions <- params$exclude_conditions
remove_norm_channel <- !is.null(params$remove_norm_channel) && params$remove_norm_channel != FALSE

if (remove_norm_channel) {
  if (is.null(exclude_conditions) || exclude_conditions == "") {
    exclude_conditions <- "Norm"
  } else {
    exclude_conditions <- paste(exclude_conditions, "Norm", sep = ",")
  }
}

if (is.null(input_file) || is.null(output_folder) || is.null(annotation_file)) {
  stop("Missing required parameters")
}
if (!dir.exists(output_folder)) dir.create(output_folder, recursive = TRUE)

message("=== MSqRob2 DIA Analysis ===")

message("\n[1/7] Reading data...")
input_sep <- detect_delimiter(input_file)
peptide_data <- read.table(input_file, sep = input_sep, header = TRUE,
                           na.strings = c("NA", "NaN", "N/A", "#VALUE!", ""),
                           check.names = FALSE, stringsAsFactors = FALSE)

message("[2/7] Reading annotation...")
annot_sep <- detect_delimiter(annotation_file)
annotation <- read.table(annotation_file, sep = annot_sep, header = TRUE,
                        stringsAsFactors = FALSE, check.names = FALSE)

colnames(peptide_data) <- make.names(colnames(peptide_data))
annotation$Sample <- make.names(annotation$Sample)
feature_id_col <- make.names(feature_id_col)
protein_col <- make.names(protein_col)

samples <- annotation$Sample
sample_cols <- intersect(samples, colnames(peptide_data))
if (length(sample_cols) == 0) stop("No matching sample columns")

message("  - Technical filtering: columns with >", col_filter * 100, "% missing values...")
keep_sample_cols <- c()
for (col_name in sample_cols) {
  na_prop <- sum(is.na(peptide_data[[col_name]])) / nrow(peptide_data)
  if (na_prop < col_filter) keep_sample_cols <- c(keep_sample_cols, col_name)
}
if (length(keep_sample_cols) == 0) stop("No sample columns remaining")
sample_cols <- keep_sample_cols

message("[3/7] Creating QFeatures object...")
rownames(peptide_data) <- make.unique(as.character(peptide_data[[feature_id_col]]))
quant_cols <- which(colnames(peptide_data) %in% sample_cols)
pe <- readQFeatures(peptide_data, quantCols = quant_cols, name = "peptideRaw")

colData_df <- data.frame(
  sample = sample_cols,
  condition = factor(make.names(annotation$Condition[match(sample_cols, annotation$Sample)])),
  row.names = sample_cols
)
if ("BioReplicate" %in% colnames(annotation)) colData_df$biorep <- factor(make.names(annotation$BioReplicate[match(sample_cols, annotation$Sample)]))
colData(pe) <- DataFrame(colData_df)
pe <- pe[, rownames(colData_df)]

message("[4/7] Filtering...")
pe <- zeroIsNA(pe, i = "peptideRaw")
pe <- filterNA(pe, i = "peptideRaw", pNA = row_filter)
keep_protein <- !is.na(rowData(pe[["peptideRaw"]])[[protein_col]]) & (rowData(pe[["peptideRaw"]])[[protein_col]] != "")
pe <- pe[keep_protein, , ]

non_na_counts <- rowSums(!is.na(assay(pe[["peptideRaw"]])))
keep_features <- non_na_counts >= filter_min_identified
pe <- pe[keep_features, , ]

if (!is.null(exclude_conditions) && exclude_conditions != "") {
  excluded <- trimws(strsplit(exclude_conditions, ",")[[1]])
  samples_to_keep <- rownames(colData(pe))[!(colData(pe)$condition %in% make.names(excluded))]
  pe <- pe[, samples_to_keep]
  colData(pe)$condition <- droplevels(colData(pe)$condition)
}

# Helper functions
do_impute <- function(i, name) {
  if (!is.null(impute_method) && impute_method != "none") {
    pe <<- impute(pe, method = impute_method, i = i, name = name); return(name)
  }
  pe <<- addAssay(pe, pe[[i]], name = name); return(name)
}
do_norm <- function(i, name) {
  if (normalize_method != "none") {
    pe <<- normalize(pe, i = i, name = name, method = normalize_method); return(name)
  }
  pe <<- addAssay(pe, pe[[i]], name = name); return(name)
}
do_agg <- function(i, fcol, name) {
  agg_fun <- switch(aggregation_method, "robust" = MsCoreUtils::robustSummary, "sum" = colSums, "mean" = colMeans, "median" = matrixStats::colMedians, "iPQF" = MsCoreUtils::robustSummary, MsCoreUtils::robustSummary)
  pe <<- pe[, colnames(pe[[i]])]
  pe <<- aggregateFeatures(pe, i = i, fcol = fcol, name = name, fun = agg_fun)
}

message("[5/7] Processing...")
if (log2_transform) { pe <- logTransform(pe, base = 2, i = "peptideRaw", name = "peptideLog"); currentAssay <- "peptideLog" } else { pe <- addAssay(pe, pe[["peptideRaw"]], name = "peptideLog"); currentAssay <- "peptideLog" }

if (aggregation_order == "before") {
  if (impute_order == "before") currentAssay <- do_impute(currentAssay, "psmImp")
  pe <- addAssay(pe, pe[[currentAssay]], name = "peptide_raw")
  do_agg(currentAssay, protein_col, "protein")
  pe <- addAssay(pe, pe[["protein"]], name = "protein_raw")
  protAssay <- do_norm("protein", "proteinNorm")
  if (impute_order == "after") protAssay <- do_impute(protAssay, "proteinImp")
  finalAssay <- protAssay
} else {
  if (impute_order == "before") currentAssay <- do_impute(currentAssay, "psmImp")
  pe <- addAssay(pe, pe[[currentAssay]], name = "peptide_raw")
  currentAssay <- do_norm(currentAssay, "psmNorm")
  if (impute_order == "after") currentAssay <- do_impute(currentAssay, "psmImpAfter")
  do_agg(currentAssay, protein_col, "protein")
  pe <- addAssay(pe, pe[["protein"]], name = "protein_raw")
  finalAssay <- "protein"
}

message("[6/7] Model fitting...")
formula <- ~ 0 + condition
pe <- msqrob(object = pe, i = finalAssay, formula = formula, ridge = ridge_penalty, robust = robust_regression, maxitRob = max_iterations)

message("[7/7] Hypothesis testing...")
if (!is.null(comparison_file) && file.exists(comparison_file)) {
  comparisons <- read.table(comparison_file, sep = detect_delimiter(comparison_file), header = TRUE, stringsAsFactors = FALSE, check.names = FALSE)
} else {
  conds <- levels(colData(pe)$condition)
  comparisons <- data.frame(comparison_label = character(), condition_A = character(), condition_B = character(), stringsAsFactors = FALSE)
  for (i in 1:(length(conds) - 1)) { for (j in (i + 1):length(conds)) { comparisons <- rbind(comparisons, data.frame(comparison_label = paste0(conds[j], "_vs_", conds[i]), condition_A = conds[i], condition_B = conds[j], stringsAsFactors = FALSE)) } }
}

all_results <- list()
valid_params <- paste0("condition", levels(colData(pe)$condition))
for (i in 1:nrow(comparisons)) {
  comp_label <- comparisons$comparison_label[i]; cond_a <- make.names(comparisons$condition_A[i]); cond_b <- make.names(comparisons$condition_B[i])
  if (!(paste0("condition", cond_a) %in% valid_params && paste0("condition", cond_b) %in% valid_params)) next
  contrast <- makeContrast(paste0("condition", cond_b, " - condition", cond_a, " = 0"), parameterNames = valid_params)
  res <- hypothesisTest(object = pe, i = finalAssay, contrast = contrast)
  df <- as.data.frame(rowData(res[[finalAssay]])[[colnames(contrast)]])
  if (adjust_method != "BH") df$adjPval <- p.adjust(df$pval, method = adjust_method)
  df$comparison <- comp_label; df$condition_A <- comparisons$condition_A[i]; df$condition_B <- comparisons$condition_B[i]
  df$protein <- rownames(df)
  df$significant <- (df$adjPval < alpha) & (abs(df$logFC) >= lfc_threshold)
  all_results[[comp_label]] <- df
}

if (length(all_results) > 0) {
  combined <- do.call(rbind, all_results); combined <- combined[order(combined$pval), ]
  write.table(combined, file = file.path(output_folder, "protein_results.txt"), sep = "\t", row.names = FALSE, quote = FALSE)
  write.table(as.data.frame(assay(pe[[finalAssay]])), file = file.path(output_folder, "protein_intensity.txt"), sep = "\t", row.names = TRUE, quote = FALSE)
}

message("[QC] Generating plots...")

# Helper for ggplot boxplots
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

# --- 1. QC PLOTS ---
pdf(file.path(output_folder, "protein_qc_plots.pdf"), width = 14, height = 10)
if ("peptide_raw" %in% names(pe)) {
  p <- plot_boxplot_ggplot(pe, "peptide_raw", "Peptide Intensities (Before Norm)")
  if (!is.null(p)) print(p)
}
if ("protein_raw" %in% names(pe)) {
  p <- plot_boxplot_ggplot(pe, "protein_raw", "Protein Intensities (Before Norm)")
  if (!is.null(p)) print(p)
}
p <- plot_boxplot_ggplot(pe, currentAssay, "Peptide Intensities (Normalized)")
if (!is.null(p)) print(p)
p <- plot_boxplot_ggplot(pe, finalAssay, "Protein Intensities (Normalized)")
if (!is.null(p)) print(p)

pca_data <- t(na.omit(assay(pe[[finalAssay]])))
if (nrow(pca_data) > 2 && ncol(pca_data) > 2) {
  pca_res <- prcomp(pca_data, scale. = TRUE)
  df_pca <- as.data.frame(pca_res$x)
  df_pca$Sample <- rownames(df_pca)
  cd <- as.data.frame(colData(pe))
  cd$Sample <- rownames(cd)
  df_pca <- merge(df_pca, cd, by = "Sample")
  p_pca <- ggplot(df_pca, aes(x = PC1, y = PC2, color = condition, label = Sample)) +
    geom_point(size = 3) + geom_text(vjust = 1.5, size = 3) + theme_minimal() +
    labs(title = "PCA - Protein Level")
  print(p_pca)
}
dev.off()

# --- 2. VOLCANO PLOTS ---
pdf(file.path(output_folder, "protein_volcano_plots.pdf"), width = 14, height = 10)
for (comp in names(all_results)) {
  df_v <- all_results[[comp]]
  v_data <- df_v[!is.na(df_v$logFC) & !is.na(df_v$adjPval), ]
  if (nrow(v_data) > 0) {
    p_vol <- ggplot(v_data, aes(x = logFC, y = -log10(adjPval), color = significant)) +
      geom_point(alpha = 0.5, size = 0.8) +
      scale_color_manual(values = c("gray", "red")) +
      theme_minimal() +
      geom_hline(yintercept = -log10(alpha), linetype = "dashed", color = "blue") +
      {if(lfc_threshold > 0) geom_vline(xintercept = c(-lfc_threshold, lfc_threshold), linetype = "dashed", color = "blue")} +
      labs(title = paste("Volcano Plot:", comp))
    print(p_vol)
  }
}
dev.off()
message("Done.")