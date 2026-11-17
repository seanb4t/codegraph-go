#include "api.h"

#ifdef __cplusplus
extern "C" {
#endif
static int sq(int v);
#ifdef __cplusplus
}
#endif

static int sq(int v) { return v * v; }

int dot(const vec_t *a, const vec_t *b) { return a->x * b->x + a->y * b->y; }

int norm(const vec_t *v) { return sq(v->x) + sq(v->y); }
