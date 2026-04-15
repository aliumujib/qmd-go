#ifndef LLAMA_GO_H
#define LLAMA_GO_H

#ifdef __cplusplus
extern "C" {
#endif

typedef void* LlamaGoModel;

LlamaGoModel llama_go_load(const char* model_path, int n_ctx, int n_gpu_layers);
void llama_go_free(LlamaGoModel model);

int llama_go_embed(LlamaGoModel model, const char* text, float* embedding, int max_dims);

const char* llama_go_get_error(void);

#ifdef __cplusplus
}
#endif

#endif
