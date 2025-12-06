import os
from datetime import datetime

import click
import requests

fields = ['TY', 'TI', 'AB', 'A1', 'A2', 'A3', 'A4', 'AD', 'AN', 'AU', 'AV', 'BT', 'C1', 'C2', 'C3', 'C4', 'C5', 'C6', 'C7', 'C8', 'CA', 'CN', 'CP', 'CT', 'CY', 'DA', 'DB', 'DO', 'DP', 'ED', 'EP', 'ET', 'ID', 'IS', 'J1', 'J2', 'JA', 'JF', 'JO', 'KW', 'L1', 'L2', 'L3', 'L4', 'LA', 'LB', 'LK', 'M1', 'M2', 'M3', 'N1', 'N2', 'NV', 'OP', 'PB', 'PP', 'PY', 'RI', 'RN', 'RP', 'SE', 'SN', 'SP', 'ST', 'T1', 'T2', 'T3', 'TA', 'TT', 'U1', 'U2', 'U3', 'U4', 'U5', 'UR', 'VL', 'VO', 'Y1', 'Y2', 'ER']

def download_json(url):
    r = requests.get(url)
    return r.json()

@click.command()
@click.option("--ids", "-i", help="Project ID", required=True)
@click.option("--output_folder", "-o", help="Output file", required=False)
def main(ids, output_folder):
    i = ids.split(",")
    result = []
    for e in i:
        e = extract_json(e)
        ris_instance =f"""
TY  - GEN
ID  - {e["ID"]}
UR  - {e["UR"]}
T1  - {e["T1"]}
Y1  - {e["Y1"]}
JF  - {e["JF"]}
"""
        if "AU" in e:
            for a in e["AU"]:
                ris_instance += f"""AU  - {a}\n"""
        result.append(ris_instance)
    if output_folder:
        os.makedirs(output_folder, exist_ok=True)
        with open(os.path.join(output_folder, "protocolsio.ris"), "w") as f:
            f.write("\n".join(result))
    else:
        print("\n".join(result))



def extract_json(id):
    url = f"https://www.protocols.io/api/v1/exports/item/{id}?format=json&type_id=1&download"
    project = download_json(url)
    d = {}
    d["TY"] = "GEN"
    d["ID"] = project["id"]
    d["UR"] = project["versions"][-1]["doi"]
    d["T1"] = project["title"]
    d["AB"] = project["description"]
    d["AU"] = [a["name"] for a in project["authors"]]
    d["Y1"] = project["created_on"]
    # convert Y1 to year from unix timestamp to year
    d["Y1"] = datetime.fromtimestamp(d["Y1"]).strftime('%Y')
    d["JF"] = "protocols.io"
    return d

if __name__ == "__main__":
    main()