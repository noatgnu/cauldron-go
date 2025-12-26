process_ptm_with_fasta <- function(data, fasta_path, seq_col, protein_col, prob_col, threshold) {
  message("Processing PTM sites using FASTA database...")
  fasta <- readAAStringSet(fasta_path)
  names(fasta) <- sapply(strsplit(names(fasta), "|", fixed = TRUE), function(x) if(length(x)>1) x[2] else x[1])

  data$site_id <- NA_character_

  for (i in seq_len(nrow(data))) {
    pep_seq <- as.character(data[[seq_col]][i])
    proteins <- as.character(data[[protein_col]][i])
    prob_str <- as.character(data[[prob_col]][i])

    if (is.na(pep_seq) || is.na(proteins) || is.na(prob_str)) next

    protein_list <- unlist(strsplit(proteins, ";"))
    main_protein <- protein_list[1] # Use leading protein
    main_protein <- gsub("^sp\|^tr\|", "", main_protein)
    main_protein <- sub("\|.*", "", main_protein)

    prob_matches <- gregexpr("[STY]\(([0-9.]+)\)", prob_str, perl = TRUE)
    prob_list <- regmatches(prob_str, prob_matches)[[1]]

    if (length(prob_list) == 0) next

    site_info <- lapply(prob_list, function(x) {
      residue <- substr(x, 1, 1)
      prob_val <- as.numeric(gsub(".*\(([0-9.]+)\).*, "\1", x))
      list(residue = residue, prob = prob_val)
    })

    high_prob_sites <- which(sapply(site_info, function(x) x$prob >= threshold))
    if (length(high_prob_sites) == 0) next

    if (!main_protein %in% names(fasta)) next
    prot_seq <- as.character(fasta[[main_protein]])

    pep_pos <- gregexpr(pep_seq, prot_seq, fixed = TRUE)[[1]]
    if (pep_pos[1] == -1) next

    pep_start <- pep_pos[1]

    site_positions <- c()
    residue_counts <- list()

    for (idx in high_prob_sites) {
      residue <- site_info[[idx]]$residue

      if (is.null(residue_counts[[residue]])) {
        residue_counts[[residue]] <- 1
      } else {
        residue_counts[[residue]] <- residue_counts[[residue]] + 1
      }

      occurrence_num <- residue_counts[[residue]]
      all_positions <- gregexpr(residue, pep_seq)[[1]]

      if (all_positions[1] == -1 || occurrence_num > length(all_positions)) next

      pep_offset <- all_positions[occurrence_num]
      abs_pos <- pep_start + pep_offset - 1
      site_positions <- c(site_positions, paste0(residue, abs_pos))
    }

    if (length(site_positions) > 0) {
      gene_symbol <- main_protein
      data$site_id[i] <- paste0(gene_symbol, "_", paste(sort(site_positions), collapse = "_"))
    }
  }

  return(data$site_id)
}
