if (!requireNamespace("sva", quietly = TRUE)) {
  stop("Package 'sva' is not installed. Please install it using: install.packages('BiocManager'); BiocManager::install('sva')")
}

if (!requireNamespace("limma", quietly = TRUE)) {
  stop("Package 'limma' is not installed. Please install it using: install.packages('BiocManager'); BiocManager::install('limma')")
}

library(sva)
library(limma)

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
batch_info_file <- params$batch_info
method <- ifelse(is.null(params$method), "combat", params$method)
use_log2 <- ifelse(is.null(params$use_log2) || params$use_log2 == "true", TRUE, FALSE)
preserve_column <- ifelse(is.null(params$preserve_column), "", params$preserve_column)

if (is.null(params$sample_cols)) {
  stop("sample_cols parameter is required")
}
sample_cols <- strsplit(params$sample_cols, ",")[[1]]

if (!file.exists(batch_info_file)) {
  stop(paste("Annotation file not found:", batch_info_file))
}

if (grepl("\\.csv$", batch_info_file)) {
  batch_df <- read.csv(batch_info_file, check.names = FALSE, stringsAsFactors = FALSE)
} else if (grepl("\\.tsv$|\\.txt$", batch_info_file)) {
  batch_df <- read.delim(batch_info_file, check.names = FALSE, sep = "\t", stringsAsFactors = FALSE)
} else {
  stop(paste("Unsupported annotation file format:", batch_info_file))
}

if (ncol(batch_df) < 2) {
  stop("Annotation file must have at least 2 columns")
}

if (!("Sample" %in% colnames(batch_df))) {
  stop("Annotation file must have a 'Sample' column")
}

if (!("Batch" %in% colnames(batch_df))) {
  stop("Annotation file must have a 'Batch' column")
}

batch_df <- batch_df[batch_df$Sample %in% sample_cols, ]

if (nrow(batch_df) == 0) {
  stop("No matching samples found in annotation file")
}

if (nrow(batch_df) != length(sample_cols)) {
  missing_samples <- setdiff(sample_cols, batch_df$Sample)
  if (length(missing_samples) > 0) {
    cat("Warning: Some samples not found in annotation file:\n")
    cat(paste(missing_samples, collapse = "\n"), "\n")
  }
}

sample_cols <- batch_df$Sample
batch_info <- batch_df$Batch

preserve_info <- NULL
if (preserve_column != "" && preserve_column %in% colnames(batch_df)) {
  preserve_info <- batch_df[[preserve_column]]
  cat(paste("Will preserve column:", preserve_column, "\n"))
} else if (preserve_column != "") {
  cat(paste("Warning: Column", preserve_column, "not found in annotation file\n"))
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

if (length(sample_cols) != length(batch_info)) {
  stop(paste("Number of sample columns (", length(sample_cols), ") must match number of batch labels (", length(batch_info), ")"))
}

for (col in sample_cols) {
  if (!(col %in% colnames(data))) {
    stop(paste("Sample column not found:", col))
  }
}

cat(paste("Processing", nrow(data), "features\n"))
cat(paste("Number of samples:", length(sample_cols), "\n"))
cat(paste("Batch correction method:", method, "\n"))
cat(paste("Batch distribution:", table(batch_info), "\n"))

sample_data <- data[, sample_cols, drop = FALSE]
for (col in sample_cols) {
  sample_data[[col]] <- as.numeric(as.character(sample_data[[col]]))
}

intensity_matrix <- as.matrix(sample_data)
rownames(intensity_matrix) <- if (!is.null(rownames(data))) rownames(data) else paste0("Feature_", 1:nrow(data))

if (use_log2) {
  cat("Converting to log2 scale before batch correction...\n")
  intensity_matrix[intensity_matrix == 0] <- NA
  intensity_matrix <- log2(intensity_matrix)
}

batch <- as.factor(batch_info)

if (method == "combat") {
  cat("Running ComBat batch correction...\n")

  if (!is.null(preserve_info) && length(preserve_info) == length(sample_cols)) {
    mod <- model.matrix(~as.factor(preserve_info))
    cat(paste("Preserving", preserve_column, "column during batch correction\n"))
    cat(paste("Values:", paste(unique(preserve_info), collapse=", "), "\n"))
  } else {
    mod <- NULL
    cat("No column preservation, proceeding without covariate adjustment\n")
  }

  corrected_matrix <- tryCatch({
    ComBat(dat = intensity_matrix, batch = batch, mod = mod, par.prior = TRUE, prior.plots = FALSE)
  }, error = function(e) {
    cat("ComBat failed, trying without parametric priors...\n")
    ComBat(dat = intensity_matrix, batch = batch, mod = mod, par.prior = FALSE, prior.plots = FALSE)
  })

} else if (method == "limma") {
  cat("Running limma removeBatchEffect...\n")

  if (!is.null(preserve_info) && length(preserve_info) == length(sample_cols)) {
    design <- model.matrix(~as.factor(preserve_info))
    cat(paste("Preserving", preserve_column, "column during batch correction\n"))
    cat(paste("Values:", paste(unique(preserve_info), collapse=", "), "\n"))
  } else {
    design <- NULL
    cat("No column preservation, proceeding without design matrix\n")
  }

  corrected_matrix <- removeBatchEffect(intensity_matrix, batch = batch, design = design)

} else {
  stop(paste("Unsupported batch correction method:", method))
}

corrected_df <- as.data.frame(corrected_matrix)
colnames(corrected_df) <- sample_cols

non_sample_cols <- setdiff(colnames(data), sample_cols)
result_data <- cbind(data[, non_sample_cols, drop = FALSE], corrected_df)

if (!dir.exists(output_folder)) {
  dir.create(output_folder, recursive = TRUE)
}

output_file <- file.path(output_folder, "batch_corrected.data.txt")
write.table(result_data, file = output_file, sep = "\t", quote = FALSE, row.names = FALSE)

batch_summary <- data.frame(
  Sample = sample_cols,
  Batch = batch_info,
  stringsAsFactors = FALSE
)
summary_file <- file.path(output_folder, "batch_info.txt")
write.table(batch_summary, file = summary_file, sep = "\t", quote = FALSE, row.names = FALSE)

cat("Batch correction completed successfully\n")
cat(paste("Output saved to:", output_file, "\n"))
cat(paste("Batch info saved to:", summary_file, "\n"))
cat(paste("Method used:", method, "\n"))
cat(paste("Features processed:", nrow(result_data), "\n"))
cat(paste("Samples processed:", ncol(corrected_df), "\n"))
