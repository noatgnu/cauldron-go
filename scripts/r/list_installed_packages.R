installed_packages <- installed.packages()[, c("Package", "Version")]
write.table(installed_packages, file = "r_requirements.txt", row.names = FALSE, col.names = FALSE, sep = " ")
