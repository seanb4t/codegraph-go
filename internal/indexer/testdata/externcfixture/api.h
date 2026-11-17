#ifndef API_H
#define API_H

#ifdef __cplusplus
extern "C" {
#endif

struct vec { int x; int y; };
typedef struct vec vec_t;

int dot(const vec_t *a, const vec_t *b);
int norm(const vec_t *v);

#ifdef __cplusplus
}
#endif

#endif
