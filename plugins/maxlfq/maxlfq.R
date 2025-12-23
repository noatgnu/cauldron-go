library(iq)
library(data.table)
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

params <- parse_args(args)

file_path <- params$file_path
output_folder <- params$output_folder
protein_col <- ifelse(is.null(params$protein_col), "Protein.Group", params$protein_col)
peptide_col <- ifelse(is.null(params$peptide_col), "Precursor.Id", params$peptide_col)
sample_cols <- strsplit(params$sample_cols, ",")[[1]]
annotation_file <- params$annotation_file
data_completeness <- ifelse(is.null(params$data_completeness), 0.7, as.numeric(params$data_completeness))
use_log2 <- !is.null(params$use_log2) && (params$use_log2 == "true" || params$use_log2 == TRUE)
max_cores <- ifelse(is.null(params$max_cores), -1, as.numeric(params$max_cores))

cat(paste("Parameter use_log2 raw value:", params$use_log2, "\n"))
cat(paste("Parameter use_log2 parsed:", use_log2, "\n"))

annotation_df <- NULL
if (!is.null(annotation_file) && file.exists(annotation_file)) {
  cat(paste("Loading annotation file:", annotation_file, "\n"))
  if (grepl("\\.csv$", annotation_file)) {
    annotation_df <- read.csv(annotation_file, check.names = FALSE, stringsAsFactors = FALSE)
  } else if (grepl("\\.tsv$|\\.txt$", annotation_file)) {
    annotation_df <- read.delim(annotation_file, check.names = FALSE, sep = "\t", stringsAsFactors = FALSE)
  }

  if (!is.null(annotation_df)) {
    if (!("Sample" %in% colnames(annotation_df))) {
      cat("Warning: Annotation file missing 'Sample' column, ignoring annotations\n")
      annotation_df <- NULL
    } else {
      cat(paste("Loaded", nrow(annotation_df), "sample annotations\n"))

      annotation_df$Sample_short <- gsub(".*[\\\\/]", "", annotation_df$Sample)
      annotation_df$Sample_short <- gsub("\\.raw$", "", annotation_df$Sample_short)

      if ("Color" %in% colnames(annotation_df)) {
        cat("Using colors from annotation file\n")
      }
    }
  }
}

if (max_cores == 0) {
  setDTthreads(0)
  cat("Using all available CPU cores\n")
} else if (max_cores == -1) {
  setDTthreads(getDTthreads() - 1)
  cat(paste("Using", getDTthreads(), "CPU cores (all minus 1)\n"))
} else if (max_cores > 0) {
  setDTthreads(max_cores)
  cat(paste("Using", getDTthreads(), "CPU cores\n"))
}

if (!file.exists(file_path)) {
  stop(paste("File not found:", file_path))
}

if (grepl("\\.csv$", file_path)) {
  data <- read.csv(file_path, check.names = FALSE, stringsAsFactors = FALSE)
} else if (grepl("\\.tsv$|\\.txt$", file_path)) {
  data <- read.delim(file_path, check.names = FALSE, sep = "\t", stringsAsFactors = FALSE)
} else {
  stop(paste("Unsupported file format:", file_path))
}

if (!(protein_col %in% colnames(data))) {
  stop(paste("Protein column not found:", protein_col))
}

if (!(peptide_col %in% colnames(data))) {
  stop(paste("Peptide column not found:", peptide_col))
}

for (col in sample_cols) {
  if (!(col %in% colnames(data))) {
    stop(paste("Sample column not found:", col))
  }
}

cat(paste("Processing", nrow(data), "rows\n"))
cat(paste("Protein column:", protein_col, "\n"))
cat(paste("Peptide column:", peptide_col, "\n"))
cat(paste("Sample columns:", paste(sample_cols, collapse = ", "), "\n"))

protein_peptide_df <- data.frame(
  protein_list = data[[protein_col]],
  id = data[[peptide_col]],
  stringsAsFactors = FALSE
)

intensity_df <- data[, sample_cols, drop = FALSE]

for (col in sample_cols) {
  intensity_df[[col]] <- as.numeric(as.character(intensity_df[[col]]))
  intensity_df[[col]][is.nan(intensity_df[[col]])] <- NA
  intensity_df[[col]][intensity_df[[col]] == 0] <- NA
}

cat(paste("Filtering data with completeness threshold:", data_completeness, "\n"))
valid_counts <- rowSums(!is.na(intensity_df))
min_valid <- max(1, ceiling(length(sample_cols) * data_completeness))
keep_rows <- valid_counts >= min_valid

protein_peptide_df <- protein_peptide_df[keep_rows, , drop = FALSE]
intensity_df <- intensity_df[keep_rows, , drop = FALSE]

cat(paste("Kept", sum(keep_rows), "of", length(keep_rows), "rows (", round(100 * sum(keep_rows) / length(keep_rows), 1), "%) after filtering\n"))

cat("Running MaxLFQ normalization...\n")

peptide_ids <- as.character(protein_peptide_df$id)
protein_ids <- as.character(protein_peptide_df$protein_list)

cat("Converting data to long format for iq::fast_MaxLFQ...\n")

if (use_log2) {
  cat("Applying log2 transformation to input intensities...\n")
  for (col in colnames(intensity_df)) {
    intensity_df[[col]] <- log2(intensity_df[[col]])
  }
  cat("Data range after log2:", min(unlist(intensity_df), na.rm = TRUE), "to", max(unlist(intensity_df), na.rm = TRUE), "\n")
}

dt_intensity <- as.data.table(intensity_df)
dt_intensity$protein_list <- protein_ids
dt_intensity$id <- peptide_ids

long_data <- data.table::melt(dt_intensity,
                  id.vars = c("protein_list", "id"),
                  measure.vars = sample_cols,
                  variable.name = "sample_list",
                  value.name = "quant",
                  na.rm = FALSE)

long_data <- long_data[!is.na(quant) & quant > 0, ]
long_data$sample_list <- as.character(long_data$sample_list)

cat(paste("Prepared", nrow(long_data), "valid intensity values for MaxLFQ\n"))

protein_list_long <- long_data$protein_list
sample_list_long <- long_data$sample_list
id_long <- long_data$id
quant_long <- long_data$quant

norm_data <- list(
  protein_list = protein_list_long,
  sample_list = sample_list_long,
  id = id_long,
  quant = quant_long
)

cat("Calling iq::fast_MaxLFQ...\n")

maxlfq_result <- iq::fast_MaxLFQ(norm_data)

protein_quant <- as.data.frame(maxlfq_result$estimate, stringsAsFactors = FALSE)
protein_quant$Protein.Group <- rownames(protein_quant)
protein_quant <- protein_quant[, c("Protein.Group", setdiff(colnames(protein_quant), "Protein.Group"))]

if (!dir.exists(output_folder)) {
  dir.create(output_folder, recursive = TRUE)
}

cat("Generating boxplot before normalization...\n")

before_norm_dt <- as.data.table(intensity_df)
before_norm_dt$protein_list <- protein_ids

protein_before <- before_norm_dt[, lapply(.SD, function(x) median(x, na.rm = TRUE)),
                                  by = protein_list, .SDcols = sample_cols]

protein_before_df <- as.data.frame(protein_before)
colnames(protein_before_df)[1] <- "Protein.Group"

before_numeric_cols <- setdiff(colnames(protein_before_df), "Protein.Group")

cat(paste("Before normalization - use_log2:", use_log2, "\n"))
cat(paste("Before normalization - sample data range:",
          min(protein_before_df[[before_numeric_cols[1]]], na.rm = TRUE), "to",
          max(protein_before_df[[before_numeric_cols[1]]], na.rm = TRUE), "\n"))

plot_data_before <- reshape2::melt(protein_before_df[, before_numeric_cols],
                                    variable.name = "Sample", value.name = "Intensity")

plot_data_before <- plot_data_before[is.finite(plot_data_before$Intensity), ]

cat(paste("Before normalization data range:", min(plot_data_before$Intensity, na.rm = TRUE), "to",
          max(plot_data_before$Intensity, na.rm = TRUE), "\n"))
cat(paste("Before normalization median:", median(plot_data_before$Intensity, na.rm = TRUE), "\n"))

plot_data_before$Sample <- gsub(".*[\\\\/]", "", plot_data_before$Sample)
plot_data_before$Sample <- gsub("\\.raw$", "", plot_data_before$Sample)

unique_samples <- unique(plot_data_before$Sample)

cat(paste("Annotation df is null:", is.null(annotation_df), "\n"))
if (!is.null(annotation_df)) {
  cat(paste("Annotation df columns:", paste(colnames(annotation_df), collapse=", "), "\n"))
  cat(paste("Has Color column:", "Color" %in% colnames(annotation_df), "\n"))
}

if (!is.null(annotation_df) && "Color" %in% colnames(annotation_df)) {
  sample_color_map <- setNames(annotation_df$Color, annotation_df$Sample_short)
  cat(paste("Color map entries:", length(sample_color_map), "\n"))
  cat(paste("Sample names in data:", paste(head(unique_samples), collapse=", "), "\n"))
  cat(paste("Sample names in annotation:", paste(head(annotation_df$Sample_short), collapse=", "), "\n"))

  sample_color_vector <- sample_color_map[unique_samples]
  sample_color_vector <- sample_color_vector[!is.na(sample_color_vector)]

  cat(paste("Matched colors:", length(sample_color_vector), "\n"))

  if (length(sample_color_vector) == 0) {
    cat("Warning: No colors matched, using rainbow colors\n")
    sample_color_vector <- setNames(rainbow(length(unique_samples)), unique_samples)
  }
} else {
  sample_color_vector <- setNames(rainbow(length(unique_samples)), unique_samples)
}

plot_data_before$Sample <- factor(plot_data_before$Sample, levels = unique_samples)

p <- ggplot(plot_data_before, aes(x = Sample, y = Intensity, fill = Sample)) +
  geom_boxplot(outlier.shape = NA, alpha = 0.8) +
  scale_fill_manual(values = sample_color_vector) +
  theme_minimal() +
  theme(
    axis.text.x = element_text(angle = 90, hjust = 1, vjust = 0.5, size = 10),
    axis.text.y = element_text(size = 10),
    axis.title = element_text(size = 12, face = "bold"),
    plot.title = element_text(size = 14, face = "bold", hjust = 0.5),
    legend.position = "none",
    panel.grid.major = element_line(color = "grey90"),
    panel.grid.minor = element_line(color = "grey95")
  ) +
  labs(
    title = "Protein Intensities Before Normalization",
    x = "Sample",
    y = "Intensity"
  )

ggsave(file.path(output_folder, "boxplot_before_normalization.svg"), plot = p, width = 14, height = 6, dpi = 300)

cat("Generating boxplot after normalization...\n")
if (use_log2) {
  cat("Data is in log2 space (transformed before MaxLFQ)\n")
} else {
  cat("Data is in raw intensity space (no log2 transformation)\n")
}

numeric_cols <- setdiff(colnames(protein_quant), "Protein.Group")
after_plot_data <- protein_quant[, c("Protein.Group", numeric_cols)]

plot_data_after <- reshape2::melt(after_plot_data[, numeric_cols], variable.name = "Sample", value.name = "Intensity")
plot_data_after <- plot_data_after[is.finite(plot_data_after$Intensity), ]

cat(paste("After normalization data range:", min(plot_data_after$Intensity, na.rm = TRUE), "to",
          max(plot_data_after$Intensity, na.rm = TRUE), "\n"))
cat(paste("After normalization median:", median(plot_data_after$Intensity, na.rm = TRUE), "\n"))

plot_data_after$Sample <- gsub(".*[\\\\/]", "", plot_data_after$Sample)
plot_data_after$Sample <- gsub("\\.raw$", "", plot_data_after$Sample)

unique_samples_after <- unique(plot_data_after$Sample)

if (!is.null(annotation_df) && "Color" %in% colnames(annotation_df)) {
  sample_color_map <- setNames(annotation_df$Color, annotation_df$Sample_short)
  cat(paste("After norm - Color map entries:", length(sample_color_map), "\n"))
  cat(paste("After norm - Sample names in data:", paste(head(unique_samples_after), collapse=", "), "\n"))

  sample_color_vector_after <- sample_color_map[unique_samples_after]
  sample_color_vector_after <- sample_color_vector_after[!is.na(sample_color_vector_after)]

  cat(paste("After norm - Matched colors:", length(sample_color_vector_after), "\n"))

  if (length(sample_color_vector_after) == 0) {
    cat("Warning: No colors matched for after plot, using rainbow colors\n")
    sample_color_vector_after <- setNames(rainbow(length(unique_samples_after)), unique_samples_after)
  }
} else {
  sample_color_vector_after <- setNames(rainbow(length(unique_samples_after)), unique_samples_after)
}

plot_data_after$Sample <- factor(plot_data_after$Sample, levels = unique_samples_after)

p_after <- ggplot(plot_data_after, aes(x = Sample, y = Intensity, fill = Sample)) +
  geom_boxplot(outlier.shape = NA, alpha = 0.8) +
  scale_fill_manual(values = sample_color_vector_after) +
  theme_minimal() +
  theme(
    axis.text.x = element_text(angle = 90, hjust = 1, vjust = 0.5, size = 10),
    axis.text.y = element_text(size = 10),
    axis.title = element_text(size = 12, face = "bold"),
    plot.title = element_text(size = 14, face = "bold", hjust = 0.5),
    legend.position = "none",
    panel.grid.major = element_line(color = "grey90"),
    panel.grid.minor = element_line(color = "grey95")
  ) +
  labs(
    title = "Protein Intensities After Normalization",
    x = "Sample",
    y = "Intensity"
  )

ggsave(file.path(output_folder, "boxplot_after_normalization.svg"), plot = p_after, width = 14, height = 6, dpi = 300)

output_file <- file.path(output_folder, "maxlfq.data.txt")
write.table(protein_quant, file = output_file, sep = "\t", quote = FALSE, row.names = FALSE)

cat("MaxLFQ normalization completed successfully\n")
cat(paste("Output saved to:", output_file, "\n"))
cat(paste("Number of proteins quantified:", nrow(protein_quant), "\n"))
cat(paste("Number of samples:", ncol(protein_quant) - 1, "\n"))
