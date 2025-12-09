library(corrplot)

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
index_col <- params$index_col
sample_cols <- strsplit(params$sample_cols, ",")[[1]]
method <- ifelse(is.null(params$method), "pearson", params$method)
min_value <- if (!is.null(params$min_value)) as.numeric(params$min_value) else NULL
order_method <- ifelse(is.null(params$order), "hclust", params$order)
hclust_method <- ifelse(is.null(params$hclust_method), "ward.D", params$hclust_method)
presenting_method <- ifelse(is.null(params$presenting_method), "ellipse", params$presenting_method)
cor_shape <- ifelse(is.null(params$cor_shape), "upper", params$cor_shape)
plot_only <- !is.null(params$plot_only) && params$plot_only == "true"
color_palette_str <- ifelse(is.null(params$color_ramp_palette),
                            "#053061,#2166AC,#4393C3,#92C5DE,#D1E5F0,#FFFFFF,#FDDBC7,#F4A582,#D6604D,#B2182B,#67001F",
                            params$color_ramp_palette)
plot_width <- ifelse(is.null(params$plot_width), 10, as.numeric(params$plot_width))
plot_height <- ifelse(is.null(params$plot_height), 10, as.numeric(params$plot_height))
text_label_size <- ifelse(is.null(params$text_label_size), 1.0, as.numeric(params$text_label_size))
number_label_size <- ifelse(is.null(params$number_label_size), 1.0, as.numeric(params$number_label_size))
label_rotation <- ifelse(is.null(params$label_rotation), 45, as.numeric(params$label_rotation))
show_diagonal <- ifelse(is.null(params$show_diagonal), TRUE, params$show_diagonal == "true")
add_grid <- ifelse(is.null(params$add_grid), FALSE, params$add_grid == "true")
grid_color <- ifelse(is.null(params$grid_color), "white", params$grid_color)
number_digits <- ifelse(is.null(params$number_digits), 2, as.numeric(params$number_digits))
plot_title <- ifelse(is.null(params$plot_title), "", params$plot_title)

if (!file.exists(file_path)) {
  stop(paste("File not found:", file_path))
}

if (grepl("\\.csv$", file_path)) {
  data <- read.csv(file_path, check.names = FALSE)
} else if (grepl("\\.tsv$|\\.txt$", file_path)) {
  data <- read.delim(file_path, check.names = FALSE, sep = "\t")
} else {
  stop(paste("Unsupported file format:", file_path))
}

if (!(index_col %in% colnames(data))) {
  stop(paste("Index column not found:", index_col))
}

for (col in sample_cols) {
  if (!(col %in% colnames(data))) {
    stop(paste("Sample column not found:", col))
  }
}

rownames(data) <- data[[index_col]]
data <- data[, sample_cols, drop = FALSE]

for (col in colnames(data)) {
  data[[col]] <- as.numeric(as.character(data[[col]]))
}

if (!plot_only) {
  data <- cor(data, method = method, use = "complete.obs")
}

data[is.na(data)] <- 1

if (is.null(min_value)) {
  min_value <- round(min(data, na.rm = TRUE), 1) - 0.1
}
max_value <- 1

color_palette <- strsplit(color_palette_str, ",")[[1]]
col <- colorRampPalette(color_palette)(200)

output_plot <- file.path(output_folder, "correlation_matrix.pdf")
output_file <- file.path(output_folder, "correlation_matrix.txt")

grid_col_param <- if (add_grid) grid_color else NA
title_param <- if (plot_title != "") plot_title else NULL

pdf(output_plot, width = plot_width, height = plot_height)

if (order_method == "hclust") {
  cor_result <- corrplot(
    as.matrix(data),
    order = order_method,
    hclust.method = hclust_method,
    method = presenting_method,
    type = cor_shape,
    is.corr = FALSE,
    col.lim = c(min_value, max_value),
    col = col,
    tl.cex = text_label_size,
    number.cex = number_label_size,
    tl.srt = label_rotation,
    diag = show_diagonal,
    addgrid.col = grid_col_param,
    number.digits = number_digits,
    title = title_param
  )
} else {
  cor_result <- corrplot(
    as.matrix(data),
    order = order_method,
    method = presenting_method,
    type = cor_shape,
    is.corr = FALSE,
    col.lim = c(min_value, max_value),
    col = col,
    tl.cex = text_label_size,
    number.cex = number_label_size,
    tl.srt = label_rotation,
    diag = show_diagonal,
    addgrid.col = grid_col_param,
    number.digits = number_digits,
    title = title_param
  )
}

dev.off()

cor_pos <- cor_result$corrPos
write.table(cor_pos, file = output_file, sep = "\t", quote = FALSE, row.names = TRUE)

cat("Correlation matrix analysis completed successfully\n")
cat(paste("Plot saved to:", output_plot, "\n"))
cat(paste("Data saved to:", output_file, "\n"))
