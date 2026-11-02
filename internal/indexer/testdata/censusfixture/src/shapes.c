#include "shapes.h"

struct point {
    int x;
    int y;
};

typedef struct point point_t;
typedef int (*area_fn)(const point_t *);

static int square(int v) { return v * v; }

int area(const point_t *p);

int area(const point_t *p) { return square(p->x) + square(p->y); }

#ifdef FAST
int scale(int v) { return v << 1; }
#else
int scale(int v) { return v * 2; }
#endif

int total(const point_t *p) { return scale(area(p)) + unused_decl(1); }
