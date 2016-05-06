# find metric lines and then remove '\' from each line
!/^-/ && !/^ *Metric/ && !/^\[Top/ && !/^ *$/ { gsub( /\\/, "", $1);  printf ("%s,\n", $1) }

