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
columns_name <- strsplit(params$columns_name, ",")[[1]]
scaler_type <- ifelse(is.null(params$scaler_type), "minmax", params$scaler_type)
with_centering <- ifelse(is.null(params$with_centering) || params$with_centering == "True", TRUE, FALSE)
with_scaling <- ifelse(is.null(params$with_scaling) || params$with_scaling == "True", TRUE, FALSE)

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

for (col in columns_name) {
  if (!(col %in% colnames(data))) {
    stop(paste("Column not found:", col))
  }
  data[[col]] <- as.numeric(as.character(data[[col]]))
}

sample_data <- data[, columns_name, drop = FALSE]

normalize_minmax <- function(x) {
  valid_x <- x[!is.na(x)]
  if (length(valid_x) == 0) return(x)
  min_val <- min(valid_x, na.rm = TRUE)
  max_val <- max(valid_x, na.rm = TRUE)
  if (min_val == max_val) return(x)
  (x - min_val) / (max_val - min_val)
}

normalize_standard <- function(x, center = TRUE, scale = TRUE) {
  if (!center && !scale) return(x)
  valid_x <- x[!is.na(x)]
  if (length(valid_x) == 0) return(x)

  mean_val <- if (center) mean(valid_x, na.rm = TRUE) else 0
  sd_val <- if (scale) sd(valid_x, na.rm = TRUE) else 1

  if (sd_val == 0) return(x)
  (x - mean_val) / sd_val
}

normalize_robust <- function(x, center = TRUE, scale = TRUE) {
  if (!center && !scale) return(x)
  valid_x <- x[!is.na(x)]
  if (length(valid_x) == 0) return(x)

  median_val <- if (center) median(valid_x, na.rm = TRUE) else 0
  iqr_val <- if (scale) IQR(valid_x, na.rm = TRUE) else 1

  if (iqr_val == 0) return(x)
  (x - median_val) / iqr_val
}

normalize_maxabs <- function(x) {
  valid_x <- x[!is.na(x)]
  if (length(valid_x) == 0) return(x)
  max_abs <- max(abs(valid_x), na.rm = TRUE)
  if (max_abs == 0) return(x)
  x / max_abs
}

normalized_data <- sample_data

if (scaler_type == "minmax") {
  for (col in columns_name) {
    normalized_data[[col]] <- normalize_minmax(sample_data[[col]])
  }
} else if (scaler_type == "standard") {
  for (col in columns_name) {
    normalized_data[[col]] <- normalize_standard(sample_data[[col]], center = with_centering, scale = with_scaling)
  }
} else if (scaler_type == "robust") {
  for (col in columns_name) {
    normalized_data[[col]] <- normalize_robust(sample_data[[col]], center = with_centering, scale = with_scaling)
  }
} else if (scaler_type == "maxabs") {
  for (col in columns_name) {
    normalized_data[[col]] <- normalize_maxabs(sample_data[[col]])
  }
} else {
  stop(paste("Unsupported scaler type:", scaler_type))
}

data[, columns_name] <- normalized_data

if (!dir.exists(output_folder)) {
  dir.create(output_folder, recursive = TRUE)
}

output_file <- file.path(output_folder, "normalized.data.txt")
write.table(data, file = output_file, sep = "\t", quote = FALSE, row.names = FALSE)

cat("Normalization completed successfully\n")
cat(paste("Output saved to:", output_file, "\n"))
cat(paste("Scaler type:", scaler_type, "\n"))
cat(paste("Columns normalized:", paste(columns_name, collapse = ", "), "\n"))
