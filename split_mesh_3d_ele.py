import csv

with open('mesh_3d_elevation.csv', 'r') as f:
    reader = csv.DictReader(f)
    for row in reader:
        meshcode = row['meshcode']
        elevation = row['elevation']
        mesh1d = meshcode[:4]
        mesh23d = meshcode[4:]
        with open(f'mesh_3d_ele_{mesh1d}.csv', 'a') as output_file:
            writer = csv.writer(output_file)
            if output_file.tell() == 0:
                # ファイルが空の場合、ヘッダーを書き込む
                writer.writerow(['mesh23d', 'elevation'])
            writer.writerow([mesh23d, elevation])
