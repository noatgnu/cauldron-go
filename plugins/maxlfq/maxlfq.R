library(iq)

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
min_samples <- ifelse(is.null(params$min_samples), 1, as.integer(params$min_samples))
use_log2 <- ifelse(is.null(params$use_log2) || params$use_log2 == "true", TRUE, FALSE)
normalize <- ifelse(is.null(params$normalize) || params$normalize == "true", TRUE, FALSE)

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
}

maxlfq_data <- cbind(annotation_df, intensity_df)

cat("Running MaxLFQ normalization...\n")
result <- iq::fast_MaxLFQ(
  maxlfq_data,
  row_names = "id",
  col_names = sample_cols,
  min_samples_observed = min_samples
)

protein_quant <- result$estimate

if (normalize) {
  cat("Applying median normalization...\n")
  for (col in colnames(protein_quant)) {
    col_data <- protein_quant[[col]]
    valid_data <- col_data[!is.na(col_data) & is.finite(col_data)]
    if (length(valid_data) > 0) {
      median_val <- median(valid_data, na.rm = TRUE)
      global_median <- median(unlist(protein_quant), na.rm = TRUE)
      protein_quant[[col]] <- col_data - median_val + global_median
    }
  }
}

if (use_log2) {
  cat("Converting to log2 scale...\n")
  for (col in colnames(protein_quant)) {
    protein_quant[[col]] <- log2(protein_quant[[col]])
  }
}

protein_quant$Protein.Group <- rownames(protein_quant)
protein_quant <- protein_quant[, c("Protein.Group", setdiff(colnames(protein_quant), "Protein.Group"))]

if (!dir.exists(output_folder)) {
  dir.create(output_folder, recursive = TRUE)
}

output_file <- file.path(output_folder, "maxlfq.data.txt")
write.table(protein_quant, file = output_file, sep = "\t", quote = FALSE, row.names = FALSE)

cat("MaxLFQ normalization completed successfully\n")
cat(paste("Output saved to:", output_file, "\n"))
cat(paste("Number of proteins quantified:", nrow(protein_quant), "\n"))
cat(paste("Number of samples:", ncol(protein_quant) - 1, "\n"))
