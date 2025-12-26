perform_pathway_enrichment <- function(differential_sites, organism, output_folder, comparison_name = NULL, alpha = 0.05, lfc_threshold = 1) {
  message("    - Performing Reactome pathway enrichment...")

  tryCatch({
    require(reactome.db, quietly = TRUE)
    pathway_to_genes <- as.list(reactomePATHID2EXTID)
    pathway_names <- as.list(reactomePATHID2NAME)

    if (organism == "human") {
      human_pathways <- pathway_to_genes[grep("^R-HSA-", names(pathway_to_genes))]
      pathways <- human_pathways
      message("    - Loaded Reactome database: ", length(pathways), " human pathways")
    } else if (organism == "mouse") {
      mouse_pathways <- pathway_to_genes[grep("^R-MMU-", names(pathway_to_genes))]
      pathways <- mouse_pathways
      message("    - Loaded Reactome database: ", length(pathways), " mouse pathways")
    } else {
      message("    - Organism '", organism, "' not supported for Reactome enrichment")
      return(NULL)
    }

    if (length(pathways) == 0) {
      message("    - No pathways found for organism: ", organism)
      return(NULL)
    }
  }, error = function(e) {
    message("    - Could not load Reactome database: ", e$message)
    return(NULL)
  })

  tryCatch({

    if (nrow(differential_sites) == 0) {
      message("    - No differential sites for pathway enrichment")
      return(NULL)
    }

    if ("adjPval" %in% colnames(differential_sites) && !"adj.P.Val" %in% colnames(differential_sites)) {
      differential_sites$adj.P.Val <- differential_sites$adjPval
    }

    valid_stats <- !is.na(differential_sites$adj.P.Val) & !is.na(differential_sites$logFC)
    sig_sites <- differential_sites[valid_stats & differential_sites$adj.P.Val < alpha & abs(differential_sites$logFC) > lfc_threshold, ]

    if (nrow(sig_sites) == 0) {
      message("  - No significant sites for pathway enrichment (alpha=", alpha, ", lfc_threshold=", lfc_threshold, ")")
      return(NULL)
    }

    message("  - Found ", nrow(sig_sites), " significant sites out of ", nrow(differential_sites), " total")

    org_db <- NULL
    if (organism == "human") {
      require(org.Hs.eg.db, quietly = TRUE)
      org_db <- org.Hs.eg.db
    } else if (organism == "mouse") {
      require(org.Mm.eg.db, quietly = TRUE)
      org_db <- org.Mm.eg.db
    } else {
      message("    - Organism '", organism, "' not supported for gene ID mapping")
      return(NULL)
    }

    extract_genes <- function(sites_df) {
      if ("Gene" %in% colnames(sites_df)) {
        genes <- unique(sites_df$Gene[!is.na(sites_df$Gene)])
      } else if ("PhosR_ID" %in% colnames(sites_df)) {
        genes <- unique(sapply(sites_df$PhosR_ID, function(id) {
          strsplit(as.character(id), "_")[[1]][1]
        }))
        genes <- genes[!is.na(genes)]
      } else {
        genes <- character(0)
      }
      return(genes)
    }

    sig_genes <- extract_genes(sig_sites)
    all_genes <- extract_genes(differential_sites)

    message("  - Extracted ", length(sig_genes), " unique genes from significant sites")
    message("  - Extracted ", length(all_genes), " unique genes from all sites")

    sig_entrez <- tryCatch({
      mapIds(org_db, sig_genes, 'ENTREZID', 'SYMBOL', multiVals = "first")
    }, error = function(e) {
      message("    - Warning: Could not map genes to Entrez IDs: ", e$message)
      return(NULL)
    })

    all_entrez <- tryCatch({
      mapIds(org_db, all_genes, 'ENTREZID', 'SYMBOL', multiVals = "first")
    }, error = function(e) {
      return(NULL)
    })

    if (is.null(sig_entrez) || is.null(all_entrez)) {
      message("    - Gene ID mapping failed, cannot perform pathway enrichment")
      return(NULL)
    }

    sig_entrez_clean <- sig_entrez[!is.na(sig_entrez)]
    all_entrez_clean <- all_entrez[!is.na(all_entrez)]

    message("  - Mapped to ", length(sig_entrez_clean), " Entrez IDs (sig) and ", length(all_entrez_clean), " Entrez IDs (all)")

    pathway_results <- list()

    for (pathway_id in names(pathways)) {
      pathway_genes <- pathways[[pathway_id]]

      sig_in_pathway <- sum(sig_entrez_clean %in% pathway_genes)
      bg_in_pathway <- sum(all_entrez_clean %in% pathway_genes)

      if (bg_in_pathway > 0) {
        fisher_result <- fisher.test(matrix(c(
          sig_in_pathway,
          length(sig_entrez_clean) - sig_in_pathway,
          bg_in_pathway - sig_in_pathway,
          length(all_entrez_clean) - bg_in_pathway
        ), nrow = 2))

        pathway_display_name <- if (pathway_id %in% names(pathway_names)) {
          pathway_names[[pathway_id]]
        } else {
          pathway_id
        }

        pathway_results[[pathway_id]] <- data.frame(
          PathwayID = pathway_id,
          Pathway = pathway_display_name,
          SignificantGenes = sig_in_pathway,
          PathwaySize = bg_in_pathway,
          PValue = fisher_result$p.value,
          OddsRatio = fisher_result$estimate,
          stringsAsFactors = FALSE
        )
      }
    }

    if (length(pathway_results) > 0) {
      pathway_df <- do.call(rbind, pathway_results)
      pathway_df$AdjPValue <- p.adjust(pathway_df$PValue, method = "BH")
      pathway_df <- pathway_df[order(pathway_df$PValue), ]
      pathway_df$Comparison <- comparison_name

      message("    - Pathway enrichment found ", nrow(pathway_df), " pathways for ", comparison_name)

      return(pathway_df)
    }

    return(NULL)
  }, error = function(e) {
    message("  - Warning: Pathway enrichment failed: ", e$message)
    return(NULL)
  })
}

generate_pathway_plots <- function(all_pathway_results, output_folder, top_n = 20) {
  if (is.null(all_pathway_results) || length(all_pathway_results) == 0) {
    return(NULL)
  }

  tryCatch({
    pdf(file.path(output_folder, "pathway_enrichment.pdf"), width = 12, height = 10)

    for (comp_name in names(all_pathway_results)) {
      pathway_df <- all_pathway_results[[comp_name]]

      if (is.null(pathway_df) || nrow(pathway_df) == 0) next

      sig_pathways <- pathway_df[pathway_df$AdjPValue < 0.05, ]

      if (nrow(sig_pathways) > 0) {
        plot_data <- head(sig_pathways[order(sig_pathways$AdjPValue), ], top_n)
        plot_data$NegLogPValue <- -log10(plot_data$AdjPValue)
        plot_data$Pathway <- factor(plot_data$Pathway, levels = rev(plot_data$Pathway))

        p <- ggplot(plot_data, aes(x = NegLogPValue, y = Pathway, fill = SignificantGenes)) +
          geom_bar(stat = "identity") +
          scale_fill_gradient(low = "lightblue", high = "darkblue") +
          theme_minimal() +
          theme(axis.text.y = element_text(size = 9)) +
          ggtitle(paste0(comp_name, ": Top ", nrow(plot_data), " Enriched Pathways")) +
          xlab("-Log10(Adjusted P-value)") +
          ylab("Pathway") +
          geom_vline(xintercept = -log10(0.05), linetype = "dashed", color = "red")
        print(p)

        if (nrow(sig_pathways) > 1) {
          p2 <- ggplot(plot_data, aes(x = PathwaySize, y = SignificantGenes,
                                      size = NegLogPValue, color = NegLogPValue, label = Pathway)) +
            geom_point(alpha = 0.6) +
            scale_color_gradient(low = "blue", high = "red") +
            theme_minimal() +
            ggtitle(paste0(comp_name, ": Pathway Enrichment Bubble Plot")) +
            xlab("Pathway Size (Total Genes)") +
            ylab("Significant Genes") +
            labs(size = "-Log10(Adj P)", color = "-Log10(Adj P)")
          print(p2)
        }
      } else {
        plot.new()
        text(0.5, 0.5, paste0(comp_name, ": No significantly enriched pathways found"), cex = 1.5)
      }
    }

    dev.off()
    message("    - Saved pathway enrichment plots (", length(all_pathway_results), " comparisons)")
  }, error = function(e) {
    if (dev.cur() != 1) dev.off()
    message("    - Warning: Pathway plotting failed: ", e$message)
  })
}

perform_ptmset_enrichment <- function(differential_sites, organism, output_folder, comparison_name = NULL, alpha = 0.05, lfc_threshold = 1) {
  message("    - Performing PTM-SET enrichment analysis...")

  tryCatch({
    data("PhosphoSitePlus", package = "PhosR")

    if (organism == "human") {
      ks_db <- PhosphoSite.human
    } else if (organism == "mouse") {
      ks_db <- PhosphoSite.mouse
    } else {
      message("  - PTM-SET enrichment not available for organism: ", organism)
      return(NULL)
    }

    if ("adjPval" %in% colnames(differential_sites) && !"adj.P.Val" %in% colnames(differential_sites)) {
      differential_sites$adj.P.Val <- differential_sites$adjPval
    }

    valid_stats <- !is.na(differential_sites$adj.P.Val) & !is.na(differential_sites$logFC)
    sig_sites <- differential_sites[valid_stats & differential_sites$adj.P.Val < alpha & abs(differential_sites$logFC) > lfc_threshold, ]

    if (nrow(sig_sites) == 0) {
      message("  - No significant sites for PTM-SET enrichment")
      return(NULL)
    }

    sig_site_ids <- if ("PhosR_ID" %in% colnames(sig_sites)) sig_sites$PhosR_ID else rownames(sig_sites)
    all_site_ids <- if ("PhosR_ID" %in% colnames(differential_sites)) differential_sites$PhosR_ID else rownames(differential_sites)

    kinase_sets <- unique(names(ks_db))
    ptmset_results <- list()

    for (kinase in kinase_sets) {
      kinase_substrates <- ks_db[[kinase]]

      sig_in_set <- sum(sig_site_ids %in% kinase_substrates)
      bg_in_set <- sum(all_site_ids %in% kinase_substrates)

      if (bg_in_set > 0) {
        fisher_result <- fisher.test(matrix(c(
          sig_in_set,
          nrow(sig_sites) - sig_in_set,
          bg_in_set - sig_in_set,
          nrow(differential_sites) - bg_in_set
        ), nrow = 2))

        ptmset_results[[kinase]] <- data.frame(
          Kinase = kinase,
          SignificantSubstrates = sig_in_set,
          TotalSubstrates = bg_in_set,
          PValue = fisher_result$p.value,
          OddsRatio = fisher_result$estimate,
          stringsAsFactors = FALSE
        )
      }
    }

    if (length(ptmset_results) > 0) {
      ptmset_df <- do.call(rbind, ptmset_results)
      ptmset_df$AdjPValue <- p.adjust(ptmset_df$PValue, method = "BH")
      ptmset_df <- ptmset_df[order(ptmset_df$PValue), ]
      ptmset_df$Comparison <- comparison_name

      message("    - PTM-SET enrichment found ", nrow(ptmset_df), " kinase sets for ", comparison_name)

      return(ptmset_df)
    }

    return(NULL)
  }, error = function(e) {
    message("  - Warning: PTM-SET enrichment failed: ", e$message)
    return(NULL)
  })
}

generate_ptmset_plots <- function(all_ptmset_results, output_folder, top_n = 20) {
  if (is.null(all_ptmset_results) || length(all_ptmset_results) == 0) {
    return(NULL)
  }

  tryCatch({
    pdf(file.path(output_folder, "ptmset_enrichment.pdf"), width = 12, height = 10)

    for (comp_name in names(all_ptmset_results)) {
      ptmset_df <- all_ptmset_results[[comp_name]]

      if (is.null(ptmset_df) || nrow(ptmset_df) == 0) next

      sig_sets <- ptmset_df[ptmset_df$AdjPValue < 0.05, ]

      if (nrow(sig_sets) > 0) {
        plot_data <- head(sig_sets[order(sig_sets$AdjPValue), ], top_n)
        plot_data$NegLogPValue <- -log10(plot_data$AdjPValue)
        plot_data$Kinase <- factor(plot_data$Kinase, levels = rev(plot_data$Kinase))

        p <- ggplot(plot_data, aes(x = NegLogPValue, y = Kinase, fill = SignificantSubstrates)) +
          geom_bar(stat = "identity") +
          scale_fill_gradient(low = "lightgreen", high = "darkgreen") +
          theme_minimal() +
          theme(axis.text.y = element_text(size = 9)) +
          ggtitle(paste0(comp_name, ": Top ", nrow(plot_data), " Enriched Kinase Substrate Sets")) +
          xlab("-Log10(Adjusted P-value)") +
          ylab("Kinase") +
          geom_vline(xintercept = -log10(0.05), linetype = "dashed", color = "red")
        print(p)
      } else {
        plot.new()
        text(0.5, 0.5, paste0(comp_name, ": No significantly enriched kinase substrate sets found"), cex = 1.5)
      }
    }

    dev.off()
    message("    - Saved PTM-SET enrichment plots (", length(all_ptmset_results), " comparisons)")
  }, error = function(e) {
    if (dev.cur() != 1) dev.off()
    message("    - Warning: PTM-SET plotting failed: ", e$message)
  })
}
