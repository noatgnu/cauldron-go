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
sample_cols <- strsplit(params$sample_cols, ",")[[1]]
batch_info <- strsplit(params$batch_info, ",")[[1]]
method <- ifelse(is.null(params$method), "combat", params$method)
use_log2 <- ifelse(is.null(params$use_log2) || params$use_log2 == "true", TRUE, FALSE)
preserve_group <- ifelse(is.null(params$preserve_group), "", params$preserve_group)

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

  if (preserve_group != "" && preserve_group %in% colnames(data)) {
    group_info <- data[[preserve_group]]
    if (length(group_info) == length(sample_cols)) {
      mod <- model.matrix(~as.factor(group_info))
      cat("Preserving biological group structure during batch correction\n")
    } else {
      mod <- NULL
      cat("Warning: Group info length doesn't match samples, proceeding without group preservation\n")
    }
  } else {
    mod <- NULL
  }

  corrected_matrix <- tryCatch({
    ComBat(dat = intensity_matrix, batch = batch, mod = mod, par.prior = TRUE, prior.plots = FALSE)
  }, error = function(e) {
    cat("ComBat failed, trying without parametric priors...\n")
    ComBat(dat = intensity_matrix, batch = batch, mod = mod, par.prior = FALSE, prior.plots = FALSE)
  })

} else if (method == "limma") {
  cat("Running limma removeBatchEffect...\n")

  if (preserve_group != "" && preserve_group %in% colnames(data)) {
    group_info <- data[[preserve_group]]
    if (length(group_info) == length(sample_cols)) {
      design <- model.matrix(~as.factor(group_info))
      cat("Preserving biological group structure during batch correction\n")
    } else {
      design <- NULL
      cat("Warning: Group info length doesn't match samples, proceeding without design matrix\n")
    }
  } else {
    design <- NULL
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
