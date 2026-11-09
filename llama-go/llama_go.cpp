#include "llama_go.h"
#include "llama.h"
#include <string>
#include <vector>
#include <mutex>
#include <cstring>
#include <cstdio>

thread_local std::string g_last_error;

static void set_error(const char* msg) {
    g_last_error = msg ? msg : "unknown error";
}

struct llama_go_state {
    llama_model* model;
    llama_context* ctx;
    int n_embd;
    std::mutex mu;
};

// llama.cpp logs ~6 KB of model and context info to stderr on every load. Keep errors only, so a
// stdio MCP server's stderr stays quiet (QA A-MEM-12).
static void quiet_log(enum ggml_log_level level, const char* text, void* /*user*/) {
    if (level == GGML_LOG_LEVEL_ERROR && text) {
        fputs(text, stderr);
    }
}

static std::once_flag g_log_once;

extern "C" {

LlamaGoModel llama_go_load(const char* model_path, int n_ctx, int n_gpu_layers) {
    if (!model_path) {
        set_error("model_path is NULL");
        return nullptr;
    }

    std::call_once(g_log_once, [] { llama_log_set(quiet_log, nullptr); });

    try {
        llama_model_params model_params = llama_model_default_params();
        model_params.n_gpu_layers = n_gpu_layers;

        llama_model* model = llama_model_load_from_file(model_path, model_params);
        if (!model) {
            set_error("failed to load model");
            return nullptr;
        }

        llama_context_params ctx_params = llama_context_default_params();
        ctx_params.embeddings = true;
        ctx_params.n_ctx = n_ctx > 0 ? n_ctx : 2048;
        ctx_params.n_batch = ctx_params.n_ctx;
        ctx_params.n_ubatch = ctx_params.n_ctx;

        llama_context* ctx = llama_init_from_model(model, ctx_params);
        if (!ctx) {
            llama_model_free(model);
            set_error("failed to create context");
            return nullptr;
        }

        auto* state = new llama_go_state();
        state->model = model;
        state->ctx = ctx;
        state->n_embd = llama_model_n_embd(model);

        return reinterpret_cast<LlamaGoModel>(state);
    } catch (const std::exception& e) {
        set_error(e.what());
        return nullptr;
    } catch (...) {
        set_error("unknown exception during model load");
        return nullptr;
    }
}

void llama_go_free(LlamaGoModel handle) {
    if (handle) {
        auto* state = reinterpret_cast<llama_go_state*>(handle);
        if (state->ctx) llama_free(state->ctx);
        if (state->model) llama_model_free(state->model);
        delete state;
    }
}

int llama_go_embed(LlamaGoModel handle, const char* text, float* embedding, int max_dims) {
    if (!handle || !text || !embedding || max_dims <= 0) {
        set_error("invalid parameters");
        return -1;
    }

    try {
        auto* state = reinterpret_cast<llama_go_state*>(handle);
        std::lock_guard<std::mutex> lock(state->mu);

        llama_context* ctx = state->ctx;
        const llama_vocab* vocab = llama_model_get_vocab(state->model);
        if (!vocab) {
            set_error("failed to get vocab");
            return -1;
        }

        int n_tokens = -llama_tokenize(vocab, text, (int32_t)strlen(text), nullptr, 0, true, false);
        if (n_tokens <= 0) {
            set_error("tokenization probe failed");
            return -1;
        }

        std::vector<llama_token> tokens(n_tokens);
        int actual = llama_tokenize(vocab, text, (int32_t)strlen(text), tokens.data(), (int32_t)tokens.size(), true, false);
        if (actual < 0) {
            set_error("tokenization failed");
            return -1;
        }
        n_tokens = actual;

        int n_ctx = (int)llama_n_ctx(state->ctx);
        if (n_tokens > n_ctx) {
            n_tokens = n_ctx;
        }

        llama_batch batch = llama_batch_init(n_tokens, 0, 1);
        batch.n_tokens = n_tokens;
        for (int i = 0; i < n_tokens; i++) {
            batch.token[i] = tokens[i];
            batch.pos[i] = i;
            batch.n_seq_id[i] = 1;
            batch.seq_id[i][0] = 0;
            batch.logits[i] = 1;
        }

        int ret = llama_encode(ctx, batch);
        llama_batch_free(batch);

        if (ret < 0) {
            set_error("decode failed");
            return -1;
        }

        const float* emb = llama_get_embeddings_seq(ctx, 0);
        if (!emb) {
            emb = llama_get_embeddings(ctx);
        }
        if (!emb) {
            set_error("failed to get embeddings");
            return -1;
        }

        int copy_size = (state->n_embd < max_dims) ? state->n_embd : max_dims;
        std::memcpy(embedding, emb, copy_size * sizeof(float));

        return state->n_embd;
    } catch (const std::exception& e) {
        set_error(e.what());
        return -1;
    } catch (...) {
        set_error("unknown exception during embedding");
        return -1;
    }
}

const char* llama_go_get_error(void) {
    return g_last_error.c_str();
}

}
