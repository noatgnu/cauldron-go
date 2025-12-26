process_ptm_with_fasta <- function(data, fasta_path, seq_col, protein_col, prob_col, threshold) {
  message("Processing PTM sites using FASTA database...")
  fasta <- readAAStringSet(fasta_path)
  names(fasta) <- sapply(strsplit(names(fasta), "|", fixed = TRUE), function(x) if(length(x)>1) x[2] else x[1])

  new_sites <- character(nrow(data))

  for (i in 1:nrow(data)) {
    prot_ids <- strsplit(as.character(data[[protein_col]][i]), ";")[[1]]
    prot_id <- prot_ids[1]

    seq <- data[[seq_col]][i]
    prob_str <- as.character(data[[prob_col]][i])

    if (is.na(prot_id) || !prot_id %in% names(fasta)) {
      new_sites[i] <- NA
      next
    }

    prot_seq <- as.character(fasta[[prot_id]])
    start_pos <- regexpr(seq, prot_seq, fixed = TRUE)[1]

    if (start_pos == -1) {
      new_sites[i] <- NA
      next
    }

    found_sites <- c()

    if (is.na(prob_str) || prob_str == "") {
        new_sites[i] <- "Unmodified"
        next
    }

    chars <- strsplit(prob_str, "")[[1]]
    curr_seq_idx <- 0
    j <- 1
    while (j <= length(chars)) {
      char <- chars[j]
      if (char == "(") {
        close_paren <- which(chars[j:length(chars)] == ")")[1]
        if (!is.na(close_paren)) {
          prob_val <- as.numeric(paste(chars[(j+1):(j+close_paren-2)], collapse=""))
          if (!is.na(prob_val) && prob_val >= threshold) {
             abs_pos <- start_pos + curr_seq_idx - 1
             residue <- substring(prot_seq, abs_pos, abs_pos)
             found_sites <- c(found_sites, paste0(residue, abs_pos))
          }
          j <- j + close_paren
        } else {
          j <- j + 1
        }
      } else {
        curr_seq_idx <- curr_seq_idx + 1
        j <- j + 1
      }
    }

    if (length(found_sites) > 0) {
      new_sites[i] <- paste(c(prot_id, found_sites), collapse="_")
    } else {
      new_sites[i] <- "Unmodified"
    }
  }
  return(new_sites)
}
