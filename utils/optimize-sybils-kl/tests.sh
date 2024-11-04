#!/bin/bash

OPTIMIZE_BIN="./optimize-sybils-kl"

$OPTIMIZE_BIN -9 7 -10 5 -11 4 -12 3 -16 1 -top 10
# printf "\nTap something to continue.."
# read -n 1

# ./optimization-sybils -9 4 -10 6 -11 4 -12 5 -16 1 -top 10
# printf "\nTap something to continue.."
# read -n 1

$OPTIMIZE_BIN -10 12 -11 5 -12 1 -13 2 -top 10
# printf "\nTap something to continue.."
# read -n 1

$OPTIMIZE_BIN -10 10 -11 3 -12 3 -13 3 -15 1 -top 10
# printf "\nTap something to continue.."
# read -n 1

$OPTIMIZE_BIN -10 7 -11 7 -12 3 -13 1 -14 1 -20 1 -top 10
