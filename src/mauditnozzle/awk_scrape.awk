# find metric lines and then remove '\' from each line
!/^-/ && !/^ *Metric/ && !/^ *Visit/ && !/^\[Top/ && !/^ *$/ { gsub( /\\/, "", $1);  printf ("%s,\n", $1) }

