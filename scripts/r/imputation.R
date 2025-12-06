library(VIM)
library(mice)

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
imputer_type <- ifelse(is.null(params$imputer_type), "knn", params$imputer_type)
n_neighbors <- ifelse(is.null(params$n_neighbors), 5, as.integer(params$n_neighbors))
max_iter <- ifelse(is.null(params$max_iter), 10, as.integer(params$max_iter))
simple_strategy <- ifelse(is.null(params$simple_strategy), "mean", params$simple_strategy)
fill_value <- ifelse(is.null(params$fill_value), 0.0, as.numeric(params$fill_value))

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

if (imputer_type == "simple") {
  for (col in columns_name) {
    col_data <- sample_data[[col]]
    na_indices <- is.na(col_data)

    if (any(na_indices)) {
      if (simple_strategy == "mean") {
        fill <- mean(col_data, na.rm = TRUE)
      } else if (simple_strategy == "median") {
        fill <- median(col_data, na.rm = TRUE)
      } else if (simple_strategy == "constant") {
        fill <- fill_value
      } else {
        stop(paste("Unsupported simple strategy:", simple_strategy))
      }

      sample_data[[col]][na_indices] <- fill
    }
  }
} else if (imputer_type == "knn") {
  sample_data <- kNN(sample_data, variable = columns_name, k = n_neighbors, imp_var = FALSE)
} else if (imputer_type == "iterative") {
  imputed <- mice(sample_data, m = 1, maxit = max_iter, method = "pmm", printFlag = FALSE)
  sample_data <- complete(imputed, 1)
} else {
  stop(paste("Unsupported imputer type:", imputer_type))
}

data[, columns_name] <- sample_data

if (!dir.exists(output_folder)) {
  dir.create(output_folder, recursive = TRUE)
}

output_file <- file.path(output_folder, "imputed.data.txt")
write.table(data, file = output_file, sep = "\t", quote = FALSE, row.names = FALSE)

cat("Imputation completed successfully\n")
cat(paste("Output saved to:", output_file, "\n"))
cat(paste("Imputer type:", imputer_type, "\n"))
cat(paste("Columns imputed:", paste(columns_name, collapse = ", "), "\n"))
