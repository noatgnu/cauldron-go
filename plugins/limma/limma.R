library(QFeatures)
library(limma)
library(MsCoreUtils)

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

replace_special_with_dot <- function(str) {
  gsub("[^[:alnum:]]+", ".", str)
}

run_differential_analysis <- function(input_file, output_folder, annotation_file,
                                     comparison_file, index_col, log2 = FALSE,
                                     aggregate_column = NULL, aggregate_method = "MsCoreUtils::robustSummary",
                                     col_filter = 0.7, row_filter = 0.7,
                                     impute_method = NULL, normalize_method = NULL,
                                     impute_order = "before") {

  message("Loading input data...")
  input_sep <- detect_delimiter(input_file)
  data_df <- read.table(input_file, sep = input_sep, header = TRUE,
                       na.strings = c("NA", "NaN", "N/A", "#VALUE!"),
                       check.names = FALSE, stringsAsFactors = FALSE)

  message("Loading annotation file...")
  annot_sep <- detect_delimiter(annotation_file)
  annotation_df <- read.table(annotation_file, sep = annot_sep, header = TRUE,
                             stringsAsFactors = FALSE, check.names = FALSE)

  message("Loading comparison file...")
  comp_sep <- detect_delimiter(comparison_file)
  comparison_df <- read.table(comparison_file, sep = comp_sep, header = TRUE,
                             stringsAsFactors = FALSE, check.names = FALSE)

  index_columns <- unlist(strsplit(index_col, ","))

  conditions <- c()
  samples <- c()
  sample_condition_map <- list()

  for (i in 1:nrow(annotation_df)) {
    sample <- annotation_df$Sample[i]
    condition <- annotation_df$Condition[i]
    samples <- c(samples, sample)
    if (!(condition %in% conditions)) {
      conditions <- c(conditions, condition)
    }
    sample_condition_map[[sample]] <- condition
  }

  comparisons <- list()
  for (i in 1:nrow(comparison_df)) {
    label <- comparison_df$comparison_label[i]
    comparisons[[label]] <- c(comparison_df$condition_A[i], comparison_df$condition_B[i])
  }

  message("Counting missing values...")
  missing_counts <- colSums(is.na(data_df))
  missing_props <- missing_counts / nrow(data_df)
  print(missing_counts)
  print(missing_props)

  message("Filtering columns with missing values...")
  keep_columns <- c()
  for (col_name in colnames(data_df)) {
    if (col_name %in% names(sample_condition_map)) {
      na_count <- sum(is.na(data_df[[col_name]]))
      na_percentage <- na_count / nrow(data_df)
      if (na_percentage < col_filter) {
        keep_columns <- c(keep_columns, col_name)
      }
    } else {
      keep_columns <- c(keep_columns, col_name)
    }
  }
  data_df <- data_df[, keep_columns, drop = FALSE]

  sample_columns_index <- which(colnames(data_df) %in% samples)

  clean_conditions <- c()
  clean_samples <- c()
  clean_index_columns <- c()

  for (idx_col in index_columns) {
    if (idx_col %in% colnames(data_df)) {
      clean_col <- replace_special_with_dot(idx_col)
      if (grepl("^[0-9]", clean_col)) {
        clean_col <- paste0("X", clean_col)
      }
      clean_index_columns <- c(clean_index_columns, clean_col)
    }
  }

  for (col_name in colnames(data_df)) {
    if (col_name %in% samples && col_name %in% names(sample_condition_map)) {
      cond <- replace_special_with_dot(sample_condition_map[[col_name]])
      if (grepl("^[0-9]", cond)) {
        cond <- paste0("X", cond)
      }
      clean_conditions <- c(clean_conditions, cond)
      clean_samples <- c(clean_samples, replace_special_with_dot(col_name))
    }
  }

  message("Creating QFeatures object...")
  data <- readQFeatures(data_df, ecol = sample_columns_index, name = "startingDF")
  data$group <- clean_conditions
  data$sample <- clean_samples
  currentAssayName <- "startingDF"
  data <- selectRowData(data, clean_index_columns)

  message("Filtering rows with missing values...")
  data <- zeroIsNA(data, i = seq_along(data))
  data <- filterNA(data, i = seq_along(data), pNA = row_filter)

  do_imputation <- function() {
    if (!is.null(impute_method) && impute_method != "" &&
        tolower(impute_method) != "none") {
      message(paste("Imputing missing values using:", impute_method))
      data <<- impute(data, method = impute_method, i = currentAssayName)
      currentAssayName <<- "imputedAssay"

      temp_df <- cbind(as.data.frame(rowData(data[[currentAssayName]])),
                      assay(data, currentAssayName))
      write.table(temp_df, file = file.path(output_folder, "imputed.txt"),
                 sep = "\t", row.names = TRUE, col.names = TRUE, quote = FALSE)
    } else if (!is.null(impute_method) && tolower(impute_method) == "none") {
      message("Skipping imputation (none selected)")
    }
  }

  if (impute_order == "before" || tolower(impute_order) == "before_normalization") {
    do_imputation()
  }

  if (log2) {
    message("Applying log2 transformation...")
    data <- addAssay(data, logTransform(data[[seq_along(data)[length(seq_along(data))]]]),
                    name = "log2")
    currentAssayName <- "log2"
  }

  if (!is.null(aggregate_column) && aggregate_column != "") {
    message(paste("Aggregating features by:", aggregate_column))
    clean_agg_col <- replace_special_with_dot(aggregate_column)
    if (grepl("^[0-9]", clean_agg_col)) {
      clean_agg_col <- paste0("X", clean_agg_col)
    }

    agg_fun <- eval(parse(text = aggregate_method))
    data <- aggregateFeatures(data, i = currentAssayName, fcol = clean_agg_col,
                             name = "aggregated", fun = agg_fun)
    currentAssayName <- "aggregated"

    temp_df <- cbind(as.data.frame(rowData(data[[currentAssayName]])),
                    assay(data, currentAssayName))
    write.table(temp_df, file = file.path(output_folder, "aggregated.txt"),
               sep = "\t", row.names = TRUE, col.names = TRUE, quote = FALSE)
  }

  if (!is.null(normalize_method) && normalize_method != "" &&
      tolower(normalize_method) != "none") {
    message(paste("Normalizing data using:", normalize_method))
    data <- addAssay(data, normalize(data[[seq_along(data)[length(seq_along(data))]]],
                                    method = normalize_method), name = "norm")
    currentAssayName <- "norm"

    temp_df <- cbind(as.data.frame(rowData(data[[currentAssayName]])),
                    assay(data, currentAssayName))
    write.table(temp_df, file = file.path(output_folder, "normalized.txt"),
               sep = "\t", row.names = TRUE, col.names = TRUE, quote = FALSE)
  } else if (!is.null(normalize_method) && tolower(normalize_method) == "none") {
    message("Skipping normalization (none selected)")
  }

  if (impute_order == "after" || tolower(impute_order) == "after_normalization") {
    do_imputation()
  }

  message("Preparing for limma analysis...")
  design <- model.matrix(~0 + data$group)
  colnames(design) <- gsub("data\\$group", "", colnames(design))
  fit <- lmFit(assay(data, currentAssayName), design)

  message("Running limma differential analysis...")
  all_results <- list()
  contrast_info <- data.frame(
    comparison = character(),
    condition_A = character(),
    condition_B = character(),
    contrast_formula = character(),
    direction = character(),
    stringsAsFactors = FALSE
  )

  for (comp_label in names(comparisons)) {
    message(paste("Processing comparison:", comp_label))

    condition_A <- replace_special_with_dot(comparisons[[comp_label]][1])
    if (grepl("^[0-9]", condition_A)) {
      condition_A <- paste0("X", condition_A)
    }

    condition_B <- replace_special_with_dot(comparisons[[comp_label]][2])
    if (grepl("^[0-9]", condition_B)) {
      condition_B <- paste0("X", condition_B)
    }

    contrast_formula <- paste0(condition_A, "-", condition_B)
    contrast_matrix <- makeContrasts(contrasts = contrast_formula, levels = design)

    contrast_info <- rbind(contrast_info, data.frame(
      comparison = comp_label,
      condition_A = comparisons[[comp_label]][1],
      condition_B = comparisons[[comp_label]][2],
      contrast_formula = contrast_formula,
      direction = paste0("Positive logFC: ", comparisons[[comp_label]][1], " > ", comparisons[[comp_label]][2],
                        "; Negative logFC: ", comparisons[[comp_label]][1], " < ", comparisons[[comp_label]][2]),
      stringsAsFactors = FALSE
    ))

    fit2 <- contrasts.fit(fit, contrast_matrix)
    fit2 <- eBayes(fit2)

    result <- topTable(fit2, coef = 1, adjust = "BH", number = Inf, sort.by = "none")
    result <- cbind(as.data.frame(rowData(data[[currentAssayName]])), result)
    result$comparison <- comp_label

    all_results[[comp_label]] <- result
  }

  message("Combining results...")
  combined_results <- do.call(rbind, all_results)
  rownames(combined_results) <- NULL

  dir.create(output_folder, showWarnings = FALSE, recursive = TRUE)

  output_file <- file.path(output_folder, "differential_analysis.txt")
  message(paste("Writing results to:", output_file))
  write.table(combined_results, file = output_file, sep = "\t",
             row.names = FALSE, col.names = TRUE, quote = FALSE)

  contrast_file <- file.path(output_folder, "contrast_matrix_info.txt")
  message(paste("Writing contrast matrix information to:", contrast_file))
  write.table(contrast_info, file = contrast_file, sep = "\t",
             row.names = FALSE, col.names = TRUE, quote = FALSE)

  message("Analysis complete!")
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
index_col <- params$index_col
log2_transform <- ifelse(is.null(params$log2) || params$log2 == FALSE, FALSE, TRUE)
aggregate_column <- params$aggregate_column
aggregate_method <- ifelse(is.null(params$aggregate_method), "MsCoreUtils::robustSummary", params$aggregate_method)
col_filter <- ifelse(is.null(params$col_filter), 0.7, as.numeric(params$col_filter))
row_filter <- ifelse(is.null(params$row_filter), 0.7, as.numeric(params$row_filter))
impute_method <- params$impute
normalize_method <- params$normalize
impute_order <- ifelse(is.null(params$impute_order), "before", params$impute_order)

if (is.null(input_file) || is.null(output_folder) ||
   is.null(annotation_file) || is.null(comparison_file) ||
   is.null(index_col)) {
  stop("Missing required arguments: input_file, output_folder, annotation_file, comparison_file, index_col", call. = FALSE)
}

run_differential_analysis(
  input_file = input_file,
  output_folder = output_folder,
  annotation_file = annotation_file,
  comparison_file = comparison_file,
  index_col = index_col,
  log2 = log2_transform,
  aggregate_column = aggregate_column,
  aggregate_method = aggregate_method,
  col_filter = col_filter,
  row_filter = row_filter,
  impute_method = impute_method,
  normalize_method = normalize_method,
  impute_order = impute_order
)
