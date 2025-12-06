options(repos = c(CRAN = "https://cloud.r-project.org/"))
print("Installing R packages...")

# Get the list of base packages
base_packages <- rownames(installed.packages(priority = "base"))

# Install BiocManager if not already installed
if (!requireNamespace("BiocManager", quietly = TRUE)) {
  install.packages("BiocManager", lib=Sys.getenv("R_LIBS_USER"))
}

# Read the package list
packages <- read.table("app/r_requirements.txt", stringsAsFactors = FALSE)

# Attempt to install all packages using BiocManager
for (i in 1:nrow(packages)) {
  package <- packages[i, 1]
  version <- packages[i, 2]
  if (!(package %in% base_packages) && !require(package, character.only = TRUE)) {
    tryCatch(
      {
        message(paste("Installing", package, "version", version, "using BiocManager"))
        BiocManager::install(package, dependencies = TRUE, lib=Sys.getenv("R_LIBS_USER"))
      },
      error = function(e) {
        message(paste("Failed to install", package, "using BiocManager:", e$message))
      }
    )
  }
}

# Check which packages are still not installed
not_installed <- c()
for (i in 1:nrow(packages)) {
  package <- packages[i, 1]
  if (!(package %in% base_packages) && !require(package, character.only = TRUE)) {
    not_installed <- c(not_installed, package)
  }
}

# Attempt to install the remaining packages using install.packages
for (package in not_installed) {
  tryCatch(
    {
      if (package == "Rmpi") {
        message("Installing Rmpi with special configuration arguments")
        install.packages("Rmpi", configure.args = c("ORTED=prted"), lib=Sys.getenv("R_LIBS_USER"))
      } else {
        message(paste("Attempting to install", package, "using install.packages"))
        install.packages(package, dependencies = TRUE, lib=Sys.getenv("R_LIBS_USER"))
      }
    },
    error = function(e) {
      message(paste("Failed to install", package, "using install.packages:", e$message))
    }
  )
}
