import argparse
import sys

# Define ANSI escape codes for colored output
RED = "\033[1;31m"
BLUE = "\033[1;34m"
GREEN = "\033[1;32m"
RESET = "\033[0m"

def compare_files(file1, file2, error, header_lines, skip_columns):
    discrepancy_found = False

    with open(file1, 'r') as f1, open(file2, 'r') as f2:
        lines1 = f1.readlines()
        lines2 = f2.readlines()

        headers = lines1[header_lines-1].strip().split(',')[skip_columns:]

        lines1 = lines1[header_lines:]  # Skip header lines
        lines2 = lines2[header_lines:]  # Skip header lines

        for i, (line1, line2) in enumerate(zip(lines1, lines2), start=header_lines+1):  # Start enumerating from line after the header
            # Split columns: assumes columns are comma separated
            columns1 = line1.strip().split(',')
            columns2 = line2.strip().split(',')

            # Convert to float if possible, else leave as string
            values1 = [float(x) if x.strip().replace('.', '', 1).isdigit() else x for x in columns1[skip_columns:]]
            values2 = [float(x) if x.strip().replace('.', '', 1).isdigit() else x for x in columns2[skip_columns:]]

            if len(values1) != len(values2):
                print(f"{RED}Line {i} has different number of values.{RESET}")
                print(f"{BLUE}Line in {file1}:{RESET} {line1.strip()}")
                print(f"{BLUE}Line in {file2}:{RESET} {line2.strip()}")
                discrepancy_found = True
                continue

            for j, (value1, value2) in enumerate(zip(values1, values2), start=1):
                if isinstance(value1, float) and isinstance(value2, float) and abs(value1 - value2) > error:
                    print(f"{RED}Line {i}, Column {headers[j-1]} (Value {j+skip_columns}) differs more than {error}.{RESET}")
                    print(f"{BLUE}Line in {file1}:{RESET} {','.join(columns1[:skip_columns] + [f'{GREEN}{x}{RESET}' if k==j-1 else x for k, x in enumerate(columns1[skip_columns:])])}")
                    print(f"{BLUE}Line in {file2}:{RESET} {','.join(columns2[:skip_columns] + [f'{GREEN}{x}{RESET}' if k==j-1 else x for k, x in enumerate(columns2[skip_columns:])])}")
                    discrepancy_found = True
                    break  # Skip to the next line after the first discrepancy is found

    return discrepancy_found

if __name__ == "__main__":
    parser = argparse.ArgumentParser(description='Compare two data files with a given error tolerance.')
    parser.add_argument('file1', type=str, help='First file for comparison')
    parser.add_argument('file2', type=str, help='Second file for comparison')
    parser.add_argument('--error', type=float, default=0.001, help='Error tolerance for comparison')
    parser.add_argument('--header_lines', type=int, default=1, help='Number of header lines to skip')
    parser.add_argument('--skip_columns', type=int, default=1, help='Number of columns to skip')
    
    args = parser.parse_args()

    discrepancy_found = compare_files(args.file1, args.file2, args.error, args.header_lines, args.skip_columns)
    if discrepancy_found:
        sys.exit(1)
    else:
        sys.exit(0)
