import pandas as pd
import click
import os
from uniprotparser.betaparser import UniprotParser, UniprotSequence
import io
import numpy as np

def get_uniprot_data(uniprot_ids: list[str]):
    """
    Get uniprot data from uniprot ids
    :param uniprot_ids: List of uniprot ids
    :return: pandas dataframe
    """
    parser = UniprotParser(include_isoform=True)
    data = []
    for p in parser.parse(uniprot_ids, 500):
        data.append(pd.read_csv(io.StringIO(p), sep="\t"))
    if len(data) == 0:
        return pd.DataFrame()
    elif len(data) == 1:
        return data[0]
    data = pd.concat(data)
    data = data[['Entry', 'Sequence']].drop_duplicates()
    return data

def load_fasta_library(fasta_file: str):
    """
    Load fasta library
    :param fasta_file: Fasta file
    :return: pd.DataFrame
    """
    df = []
    with open(fasta_file, 'rt') as f:
        fasta_data = ""
        current_label = ""
        for line in f:
            line = line.strip()
            if line.startswith(">"):
                if fasta_data != "":
                    df.append([current_label[:], fasta_data[:]])
                acc = UniprotSequence(line, parse_acc=True)
                if acc.accession:
                    current_label = acc.accession+acc.isoform
                else:
                    current_label = line[1:]
                fasta_data = ""
            else:
                fasta_data += line
        if fasta_data != "":
            df.append([current_label[:], fasta_data[:]])
    if len(df) == 0:
        return pd.DataFrame()
    return pd.DataFrame(df, columns=["Entry", "Sequence"]).drop_duplicates()

def process_coverage(input_file: str, value_columns: list[str], index_column: str, sequence_column: str, uniprot_acc_column: str, fasta_file: str="", output_folder: str="."):
    if input_file.endswith(".csv"):
        df = pd.read_csv(input_file)
    elif input_file.endswith(".tsv") or input_file.endswith(".txt"):
        df = pd.read_csv(input_file, sep="\t")
    else:
        raise ValueError(f"Unsupported file format: {input_file}")
    for i, r in df.iterrows():
        uni = UniprotSequence(r[uniprot_acc_column], parse_acc=True)
        if uni.accession:
            df.at[i, "MatchACC"] = uni.accession+uni.isoform
        else:
            df.at[i, "MatchACC"] = r[uniprot_acc_column]
    melted = df.melt(id_vars=[index_column, sequence_column, uniprot_acc_column, "MatchACC"], value_vars=value_columns, value_name="Value")
    melted = melted.dropna()
    melted = melted.drop_duplicates()
    melted = melted.reset_index()
    melted.drop(columns="index", inplace=True)

    if fasta_file == "":
        acc = melted["MatchACC"].unique()
        uniprot_data = get_uniprot_data(acc)
    else:
        uniprot_data = load_fasta_library(fasta_file)

    uniprot_data = uniprot_data[["Entry", "Sequence"]]
    melted = melted.merge(uniprot_data, left_on="MatchACC", right_on="Entry", how="left")
    melted = melted.drop(columns="Entry")
    coverage_dict = {}
    for i, r in melted.iterrows():
        if pd.notnull(r["Sequence"]):
            pos = r["Sequence"].index(r[sequence_column])
            if r["MatchACC"] not in coverage_dict:
                coverage_dict[r["MatchACC"]] = {}
            if pos >= 0:
                melted.at[i, "StartPos"] = pos
                end = pos + len(r[sequence_column])
                melted.at[i, "EndPos"] = end
                melted.at[i, "PeptideLength"] = len(r[sequence_column])
                r["StartPos"] = pos
                r["EndPos"] = end

                for i2 in range(pos, end):
                    if i2 not in coverage_dict[r["MatchACC"]]:
                        coverage_dict[r["MatchACC"]][i2] = []
                    coverage_dict[r["MatchACC"]][i2].append(r[index_column])
    melted["PeptideLength"] = melted["PeptideLength"].astype(int)
    melted["StartPos"] = melted["StartPos"].astype(int)
    melted["EndPos"] = melted["EndPos"].astype(int)
    result = []
    for g, d in melted.groupby(["MatchACC", "variable"]):
        d["SeqLength"] = d[sequence_column].apply(lambda x: len(x))
        d.sort_values(by="SeqLength", ascending=False, inplace=True)
        row_precursor_map = {}
        height_map = {}
        dataMap = {}
        for i, r in d.iterrows():
            row_precursor_map[r[index_column]] = r
            height = 0
            while True:
                flag_count = 0
                for i2 in range(r["StartPos"], r["EndPos"]):
                    if i2 in height_map:
                        value_array = [v for k, v in height_map[i2].items() if k != "highest"]
                        if height not in value_array:
                            flag_count += 1
                        else:
                            height += 1
                            break
                    else:
                        height_map[i2] = {"highest": 0}
                        flag_count += 1
                if flag_count == r["PeptideLength"]:
                    break
            for i2 in range(r["StartPos"], r["EndPos"]):
                height_map[i2][r[index_column]] = height
                if height > height_map[i2]["highest"]:
                    height_map[i2]["highest"] = height

                if r[index_column] not in dataMap:
                    dataMap[r[index_column]] = {
                        "x": [],
                        "y": [],
                    }
                dataMap[r[index_column]]["x"].append(i2 + 1)
                dataMap[r[index_column]]["y"].append(height_map[i2][r[index_column]])

        print(dataMap)
        for a in dataMap:
            result.append([g[0], g[1], a, dataMap[a]["x"][0]+1, dataMap[a]["x"][-1], dataMap[a]["y"][0]])
    result = pd.DataFrame(result, columns=["MatchACC", "RawFile", "Precursor", "StartPos", "EndPos", "Height"])

    result = result.merge(melted[[index_column, "MatchACC", "variable", "Value"]].drop_duplicates(), left_on=["MatchACC", "RawFile", "Precursor"], right_on=["MatchACC", "variable", index_column], how="left")
    result = result.drop(columns="variable").drop_duplicates()
    os.makedirs(output_folder, exist_ok=True)
    result.to_csv(os.path.join(output_folder, "coverage.txt"), sep="\t", index=False)
    uniprot_data.to_csv(os.path.join(output_folder, "uniprot_data.txt"), sep="\t", index=False)


        # max_value = d["Value"].max()
        # for c in height_map:
        #     if height_map[c]["highest"] > highest:
        #         highest = height_map[c]["highest"]
        # rowData = {i+1: [] for i in range(0, highest)}
        # for h in rowData:
        #     for i in range(len(d["Sequence"].values[0])):
        #         pos_data = np.nan
        #         for a in dataMap:
        #             if i+1 in dataMap[a]["x"] and h in dataMap[a]["y"]:
        #                 pos_data = row_precursor_map[a]["Value"]
        #                 break
        #         rowData[h].append(pos_data)
        # coverage_data = []
        # for h in rowData:
        #     coverage_data.append([f"{g[0]}_{h}", *rowData[h]])
        # coverage_data.append([g[0], *[max_value for i in range(len(d["Sequence"].values[0]))]])
        # coverage_data = pd.DataFrame(coverage_data, columns=["Row", *list(d["Sequence"].values[0])])
        # coverage_data.set_index("Row", inplace=True)
        #fig, ax = plt.subplots(figsize=(20, 10))
        #sns.heatmap(coverage_data, cbar=False, xticklabels=1, yticklabels=1, ax=ax, cmap="viridis")
        #fig.savefig(f"{g[0]}_{g[1]}.png")


@click.command()
@click.option("--input_file", help="Input file", type=str)
@click.option("--value_columns", help="Value columns delimited by comma", type=str)
@click.option("--index_column", help="Index column", type=str)
@click.option("--sequence_column", help="Sequence column", type=str)
@click.option("--uniprot_acc_column", help="Uniprot accession column", type=str)
@click.option("--fasta_file", help="Fasta file", default="", type=str)
@click.option("--output_folder", help="Output folder", default=".", type=str)
def main(input_file, value_columns, index_column, sequence_column, uniprot_acc_column, fasta_file, output_folder):
    process_coverage(input_file, value_columns.split(","), index_column, sequence_column, uniprot_acc_column, fasta_file, output_folder)

if __name__ == "__main__":
    main()



