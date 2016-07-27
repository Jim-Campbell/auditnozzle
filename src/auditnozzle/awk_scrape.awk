# find metric lines and then remove '\' from each line
BEGIN {origin = "CSV"}
/Default Origin Name/ { gsub( /^.*Default Origin Name: /,"", $0); gsub( /\\/,"", $0); origin=$0; next }
!/^-/ && !/^\+/ && !/^ *Metric/ && !/^ *Visit/ && !/^\[Top/ && !/^ *$/ { 	gsub( /"/, "", $0)
																gsub( /,/, "", $0)
																gsub( /\\/, "", $1);  
																printf ("%s,%s\n", origin, $1); 
																next
																}


