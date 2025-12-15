library(VennDiagram)
library(grid)

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
set_names <- if(is.null(params$set_names) || params$set_names == "") NULL else strsplit(params$set_names, ",")[[1]]
threshold <- ifelse(is.null(params$threshold), 0, as.numeric(params$threshold))
use_presence <- ifelse(is.null(params$use_presence) || params$use_presence == "true", TRUE, FALSE)
fill_colors <- ifelse(is.null(params$fill_colors), "", params$fill_colors)
alpha <- ifelse(is.null(params$alpha), 0.5, as.numeric(params$alpha))

cat("Parameters received:\n")
cat(paste("  File path:", file_path, "\n"))
cat(paste("  Sample cols:", paste(sample_cols, collapse=", "), "\n"))
cat(paste("  Set names:", ifelse(is.null(set_names), "NULL", paste(set_names, collapse=", ")), "\n"))
cat(paste("  Threshold:", threshold, "\n"))
cat(paste("  Use presence:", use_presence, "\n"))
cat(paste("  Alpha:", alpha, "\n"))

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

if (length(sample_cols) < 2 || length(sample_cols) > 5) {
  stop("Venn diagram requires 2-5 sample columns")
}

if (length(set_names) != length(sample_cols)) {
  set_names <- sample_cols
  cat("Using sample column names as set names\n")
}

for (col in sample_cols) {
  if (!(col %in% colnames(data))) {
    stop(paste("Sample column not found:", col))
  }
}

cat(paste("Processing", nrow(data), "features\n"))
cat(paste("Number of sets:", length(sample_cols), "\n"))
cat(paste("Set names:", paste(set_names, collapse = ", "), "\n"))
cat(paste("Threshold:", threshold, "\n"))

sets_list <- list()
for (i in seq_along(sample_cols)) {
  col <- sample_cols[i]
  col_data <- as.numeric(as.character(data[[col]]))

  if (use_presence) {
    valid_idx <- which(!is.na(col_data) & col_data > threshold)
  } else {
    valid_idx <- which(col_data > threshold)
  }

  if (!is.null(rownames(data))) {
    sets_list[[set_names[i]]] <- rownames(data)[valid_idx]
  } else {
    sets_list[[set_names[i]]] <- valid_idx
  }

  cat(paste("Set", set_names[i], ":", length(valid_idx), "features\n"))
}

if (fill_colors != "") {
  colors <- strsplit(fill_colors, ",")[[1]]
  if (length(colors) != length(sample_cols)) {
    cat("Warning: Number of colors doesn't match number of sets, using default colors\n")
    fill_colors <- ""
  }
} else {
  fill_colors <- ""
}

if (fill_colors == "") {
  default_colors <- c("#1f77b4", "#ff7f0e", "#2ca02c", "#d62728", "#9467bd")
  colors <- default_colors[1:length(sample_cols)]
} else {
  colors <- strsplit(fill_colors, ",")[[1]]
}

if (!dir.exists(output_folder)) {
  dir.create(output_folder, recursive = TRUE)
}

output_file_svg <- file.path(output_folder, "venn_diagram.svg")
output_file_pdf <- file.path(output_folder, "venn_diagram.pdf")

venn_obj <- venn.diagram(
  x = sets_list,
  filename = NULL,
  category.names = set_names,
  fill = colors,
  alpha = alpha,
  cex = 1.5,
  cat.cex = 1.5,
  cat.default.pos = "outer",
  margin = 0.1,
  lwd = 2,
  cat.fontface = "bold",
  fontfamily = "sans",
  disable.logging = TRUE
)

svg(output_file_svg, width = 10, height = 10)
grid.draw(venn_obj)
dev.off()

pdf(output_file_pdf, width = 10, height = 10)
grid.draw(venn_obj)
dev.off()

overlaps <- calculate.overlap(sets_list)

overlap_summary <- data.frame(
  Set = character(),
  Count = numeric(),
  stringsAsFactors = FALSE
)

for (i in seq_along(overlaps)) {
  overlap_name <- names(overlaps)[i]
  overlap_count <- length(overlaps[[i]])
  overlap_summary <- rbind(overlap_summary, data.frame(Set = overlap_name, Count = overlap_count))
}

summary_file <- file.path(output_folder, "venn_summary.txt")
write.table(overlap_summary, file = summary_file, sep = "\t", quote = FALSE, row.names = FALSE)

all_elements <- unique(unlist(sets_list))
presence_matrix <- matrix(0, nrow = length(all_elements), ncol = length(sample_cols))
rownames(presence_matrix) <- all_elements
colnames(presence_matrix) <- set_names

for (i in seq_along(sets_list)) {
  presence_matrix[sets_list[[i]], i] <- 1
}

presence_df <- as.data.frame(presence_matrix)
presence_df$Feature <- rownames(presence_df)
presence_df <- presence_df[, c("Feature", set_names)]

presence_file <- file.path(output_folder, "venn_presence.txt")
write.table(presence_df, file = presence_file, sep = "\t", quote = FALSE, row.names = FALSE)

cat("Venn diagram created successfully\n")
cat(paste("SVG output saved to:", output_file_svg, "\n"))
cat(paste("PDF output saved to:", output_file_pdf, "\n"))
cat(paste("Summary saved to:", summary_file, "\n"))
cat(paste("Presence matrix saved to:", presence_file, "\n"))
cat(paste("Total unique features:", length(all_elements), "\n"))
