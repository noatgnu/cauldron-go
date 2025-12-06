import json
import warnings
import click
import pandas as pd
import numpy

numpy.warnings = warnings

from pyclustering.cluster.center_initializer import kmeans_plusplus_initializer
from pyclustering.cluster.fcm import fcm
from sklearn.decomposition import PCA
import os


def fuzz_cluster(file_path: str, annotation_path: str, output_folder: str, center_count: list[int]):
    if not file_path:
        raise ValueError("file_path is required")
    if not annotation_path:
        raise ValueError("annotation_path is required")

    if file_path.endswith(".csv"):
        df = pd.read_csv(file_path)
    elif file_path.endswith(".tsv") or file_path.endswith(".txt"):
        df = pd.read_csv(file_path, sep="\t")
    else:
        raise ValueError("file_path should be a csv or tsv or txt file")

    if annotation_path.endswith(".csv"):
        annotation = pd.read_csv(annotation_path)
    elif annotation_path.endswith(".tsv") or annotation_path.endswith(".txt"):
        annotation = pd.read_csv(annotation_path, sep="\t")
    else:
        raise ValueError("annotation_path should be a csv or tsv or txt file")

    df_t = df[annotation["Sample"].tolist()].transpose()
    for center in center_count:
        pca = PCA(n_components=2)
        points = pca.fit_transform(df_t)
        explanation = pca.explained_variance_ratio_
        initial_centers = kmeans_plusplus_initializer(points, center,
                                                      kmeans_plusplus_initializer.FARTHEST_CENTER_CANDIDATE).initialize()
        fcm_instance = fcm(points, initial_centers)
        fcm_instance.process()
        clusters = fcm_instance.get_clusters()
        points = pd.DataFrame(points, columns=["x", "y"])
        for i in clusters:
            for j in i:
                points.at[j, "cluster"] = str(clusters.index(i))

        pd.concat([annotation, points], axis=1).to_csv(
            os.path.join(output_folder, f"fcm_{center}_clusters.txt"),
            index=False,
            sep="\t"
        )
        with open(os.path.join(output_folder, "explained_variance_ratio.json"), "w") as f:
            f.write(json.dumps(explanation.tolist()))


@click.command()
@click.option('--file_path', help='Path to the file to be processed')
@click.option('--annotation_path', help='Path to the annotation file to be processed')
@click.option('--output_folder', help='Path to the output folder')
@click.option('--center_count', help='Number of clusters to be generated', type=str)
def main(file_path: str, annotation_path: str, output_folder: str, center_count: str):
    center_count = list(map(int, center_count.split(",")))
    fuzz_cluster(file_path, annotation_path, output_folder, center_count)


if __name__ == "__main__":
    main()