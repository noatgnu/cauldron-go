library(iq)
library(data.table)

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
use_log2 <- ifelse(is.null(params$use_log2) || params$use_log2 == "true", TRUE, FALSE)
normalize <- ifelse(is.null(params$normalize) || params$normalize == "true", TRUE, FALSE)
max_cores <- ifelse(is.null(params$max_cores), -1, as.numeric(params$max_cores))

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
    } else if (!("Condition" %in% colnames(annotation_df))) {
      cat("Warning: Annotation file missing 'Condition' column, ignoring annotations\n")
      annotation_df <- NULL
    } else {
      cat(paste("Loaded", nrow(annotation_df), "sample annotations\n"))
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

annotation_df <- data.frame(
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

annotation_df <- annotation_df[keep_rows, , drop = FALSE]
intensity_df <- intensity_df[keep_rows, , drop = FALSE]

cat(paste("Kept", sum(keep_rows), "of", length(keep_rows), "rows (", round(100 * sum(keep_rows) / length(keep_rows), 1), "%) after filtering\n"))

cat("Running MaxLFQ normalization...\n")

peptide_ids <- as.character(annotation_df$id)
protein_ids <- as.character(annotation_df$protein_list)

cat("Converting data to long format for iq::fast_MaxLFQ...\n")

dt_intensity <- as.data.table(intensity_df)
dt_intensity$protein_list <- protein_ids
dt_intensity$id <- peptide_ids

long_data <- melt(dt_intensity,
                  id.vars = c("protein_list", "id"),
                  measure.vars = sample_cols,
                  variable.name = "sample_list",
                  value.name = "quant",
                  na.rm = FALSE)

long_data <- long_data[!is.na(quant) & quant > 0]
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
numeric_cols <- setdiff(colnames(protein_quant), "Protein.Group")
plot_data_before <- reshape2::melt(protein_quant[, numeric_cols], variable.name = "Sample", value.name = "Intensity")

if (!is.null(annotation_df)) {
  sample_lookup <- setNames(annotation_df$Sample, annotation_df$Sample)
  for (i in 1:nrow(plot_data_before)) {
    sample_path <- as.character(plot_data_before$Sample[i])
    sample_name <- gsub(".*[\\\\/]", "", sample_path)
    sample_name <- gsub("\\.raw$", "", sample_name)
    matching_rows <- which(annotation_df$Sample == sample_path | annotation_df$Sample == sample_name)
    if (length(matching_rows) > 0) {
      plot_data_before$Condition[i] <- annotation_df$Condition[matching_rows[1]]
    } else {
      plot_data_before$Condition[i] <- "Unknown"
    }
  }
  plot_data_before$Sample <- gsub(".*[\\\\/]", "", plot_data_before$Sample)
  plot_data_before$Sample <- gsub("\\.raw$", "", plot_data_before$Sample)

  svg(file.path(output_folder, "boxplot_before_normalization.svg"), width = 12, height = 6)
  par(mar = c(10, 4, 4, 2))
  boxplot(Intensity ~ Condition, data = plot_data_before,
          main = "Protein Intensities Before Normalization (by Condition)",
          xlab = "", ylab = "Intensity",
          las = 2, col = rainbow(length(unique(plot_data_before$Condition))), outline = FALSE)
  dev.off()
} else {
  plot_data_before$Sample <- gsub(".*[\\\\/]", "", plot_data_before$Sample)
  plot_data_before$Sample <- gsub("\\.raw$", "", plot_data_before$Sample)

  svg(file.path(output_folder, "boxplot_before_normalization.svg"), width = 12, height = 6)
  par(mar = c(10, 4, 4, 2))
  boxplot(Intensity ~ Sample, data = plot_data_before,
          main = "Protein Intensities Before Normalization",
          xlab = "", ylab = "Intensity",
          las = 2, col = "lightblue", outline = FALSE)
  dev.off()
}

if (normalize) {
  cat("Applying median normalization...\n")
  for (col in numeric_cols) {
    col_data <- protein_quant[[col]]
    valid_data <- col_data[!is.na(col_data) & is.finite(col_data)]
    if (length(valid_data) > 0) {
      median_val <- median(valid_data, na.rm = TRUE)
      global_median <- median(unlist(protein_quant[, numeric_cols]), na.rm = TRUE)
      protein_quant[[col]] <- col_data - median_val + global_median
    }
  }

  cat("Generating boxplot after normalization...\n")
  plot_data_after <- reshape2::melt(protein_quant[, numeric_cols], variable.name = "Sample", value.name = "Intensity")

  if (!is.null(annotation_df)) {
    for (i in 1:nrow(plot_data_after)) {
      sample_path <- as.character(plot_data_after$Sample[i])
      sample_name <- gsub(".*[\\\\/]", "", sample_path)
      sample_name <- gsub("\\.raw$", "", sample_name)
      matching_rows <- which(annotation_df$Sample == sample_path | annotation_df$Sample == sample_name)
      if (length(matching_rows) > 0) {
        plot_data_after$Condition[i] <- annotation_df$Condition[matching_rows[1]]
      } else {
        plot_data_after$Condition[i] <- "Unknown"
      }
    }
    plot_data_after$Sample <- gsub(".*[\\\\/]", "", plot_data_after$Sample)
    plot_data_after$Sample <- gsub("\\.raw$", "", plot_data_after$Sample)

    svg(file.path(output_folder, "boxplot_after_normalization.svg"), width = 12, height = 6)
    par(mar = c(10, 4, 4, 2))
    boxplot(Intensity ~ Condition, data = plot_data_after,
            main = "Protein Intensities After Normalization (by Condition)",
            xlab = "", ylab = "Intensity",
            las = 2, col = rainbow(length(unique(plot_data_after$Condition))), outline = FALSE)
    dev.off()
  } else {
    plot_data_after$Sample <- gsub(".*[\\\\/]", "", plot_data_after$Sample)
    plot_data_after$Sample <- gsub("\\.raw$", "", plot_data_after$Sample)

    svg(file.path(output_folder, "boxplot_after_normalization.svg"), width = 12, height = 6)
    par(mar = c(10, 4, 4, 2))
    boxplot(Intensity ~ Sample, data = plot_data_after,
            main = "Protein Intensities After Normalization",
            xlab = "", ylab = "Intensity",
            las = 2, col = "lightgreen", outline = FALSE)
    dev.off()
  }
}

if (use_log2) {
  cat("Converting to log2 scale...\n")
  numeric_cols <- setdiff(colnames(protein_quant), "Protein.Group")
  for (col in numeric_cols) {
    protein_quant[[col]] <- log2(protein_quant[[col]])
  }
}

output_file <- file.path(output_folder, "maxlfq.data.txt")
write.table(protein_quant, file = output_file, sep = "\t", quote = FALSE, row.names = FALSE)

cat("MaxLFQ normalization completed successfully\n")
cat(paste("Output saved to:", output_file, "\n"))
cat(paste("Number of proteins quantified:", nrow(protein_quant), "\n"))
cat(paste("Number of samples:", ncol(protein_quant) - 1, "\n"))
