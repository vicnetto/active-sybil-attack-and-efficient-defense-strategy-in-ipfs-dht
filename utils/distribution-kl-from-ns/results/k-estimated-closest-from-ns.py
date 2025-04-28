import numpy as np
import matplotlib.pyplot as plt
import pandas as pd
import os
import sys

FONT_SIZE = 15

def plot_fitted_curve(data_file):
    # Read the data
    data = pd.read_csv(data_file, delimiter=';', header=None, names=['x', 'y'])
    x_values = data['x'].values
    y_values = data['y'].values
    
    # Fit a polynomial curve
    coefficients = np.polyfit(x_values, y_values, deg=3)
    polynomial = np.poly1d(coefficients)
    
    # Generate smooth x values for plotting
    x_smooth = np.linspace(min(x_values), max(x_values), 500)
    y_smooth = polynomial(x_smooth)
    
    # Determine output file name
    output_file = os.path.splitext(data_file)[0] + ".pdf"
    
    # Plot the fitted curve
    plt.figure(figsize=(9, 5))
    plt.plot(x_smooth, y_smooth, color='darkcyan')
    plt.axhline(y=0.94, color='red', linestyle='--', label='$threshold = 0.94$')
    
    # Highlight specific points with vertical and horizontal lines
    points = [
        (12347, 0.915344798766758, 'green', '$ns_{minimum}$', ':'),
        (13239, 0.846563185122119, 'purple', rf'$ns_{{average}}${'\n'}', '-.'),
        # (12043, 0.940048973173594, 'blue', rf'{'\n'}       $\approx 90\%$ of $ns_{{average}}$', '-'),
        ]
    for x_p, y_p, color, label, linestyle in points:
        plt.scatter(x_p, y_p, color=color, zorder=3)
        # plt.plot([x_p, x_p], [min(y_values), y_p], color=color, linestyle=linestyle, alpha=0.7, label=label)
        # plt.plot([min(x_values), x_p], [y_p, y_p], color=color, linestyle=linestyle, alpha=0.7)
        if x_p == 13239:
            plt.text(x_p + 50, y_p, rf'{label} = ({x_p}, $\approx 0.850$)', fontsize=FONT_SIZE, color=color)
            continue
        plt.text(x_p + 50, y_p, f'{label} = ({x_p}, {y_p:.3f})', fontsize=FONT_SIZE, color=color)

    ten_percent_x = 12043
    ten_percent_y = 0.940048973173594
    plt.scatter(ten_percent_x, ten_percent_y, color='blue', zorder=3)
    plt.text(ten_percent_x + 30, ten_percent_y + 0.015, rf'$\approx 90\% \times ns_{{average}}$ = ({ten_percent_x}, {ten_percent_y:.3f})', fontsize=FONT_SIZE, color='blue')
    # plt.plot([ten_percent_x, ten_percent_x], [min(y_values), ten_percent_y], color='blue', linestyle='--')
    
    plt.xlabel('Network Size Estimation', size=FONT_SIZE)
    plt.xlim(min(x_values), max(x_values))
    plt.ylim(min(y_values), max(y_values))
    plt.ylabel('$D_{KL}$', size=FONT_SIZE)
    plt.legend(prop={'size': FONT_SIZE})
    plt.xticks(size=FONT_SIZE)
    plt.yticks(size=FONT_SIZE)
    plt.tight_layout()
    plt.grid()
    
    # Save the plot as a PDF
    pdf_filename = os.path.splitext(data_file)[0] + '.pdf'
    plt.savefig(pdf_filename, format='pdf')
    plt.close()

    print("Plot exported to", pdf_filename)

if __name__ == "__main__":
    if len(sys.argv) < 2:
        print("Usage: python", sys.argv[0] ,"script.py <csv_filename> [interval]")
        sys.exit(1)

    csv_filename = sys.argv[1]
    interval = float(sys.argv[2]) if len(sys.argv) > 2 else 0.1

    plot_fitted_curve(csv_filename)
