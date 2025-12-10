#include <stdio.h>

// Simple runtime for extern functions used in tds25.ctds
long get_int(void) {
    long v = 0;
    if (scanf("%ld", &v) != 1) {
        return 0;
    }
    return v;
}

long print_int(long i) {
    printf("The number is: %ld\n", i);
    return i;
}

long print_bool(long b) {
    const char *s = (b != 0) ? "true" : "false";
    printf("The boolean is: %s\n", s);
    return b;
}



