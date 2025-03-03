package api_config

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"gopkg.in/yaml.v3"
)

type Config struct {
	PredictionOptions `yaml:"parameters"`
	Name              string `yaml:"name"`

	F16            bool              `yaml:"f16"`
	Threads        int               `yaml:"threads"`
	Debug          bool              `yaml:"debug"`
	Roles          map[string]string `yaml:"roles"`
	Embeddings     bool              `yaml:"embeddings"`
	Backend        string            `yaml:"backend"`
	TemplateConfig TemplateConfig    `yaml:"template"`

	PromptStrings, InputStrings                []string `yaml:"-"`
	InputToken                                 [][]int  `yaml:"-"`
	functionCallString, functionCallNameString string   `yaml:"-"`

	FunctionsConfig Functions `yaml:"function"`

	FeatureFlag FeatureFlag `yaml:"feature_flags"` // Feature Flag registry. We move fast, and features may break on a per model/backend basis. Registry for (usually temporary) flags that indicate aborting something early.
	// LLM configs (GPT4ALL, Llama.cpp, ...)
	LLMConfig `yaml:",inline"`

	// AutoGPTQ specifics
	AutoGPTQ AutoGPTQ `yaml:"autogptq"`

	// Diffusers
	Diffusers Diffusers `yaml:"diffusers"`

	Step int `yaml:"step"`

	// GRPC Options
	GRPC GRPC `yaml:"grpc"`

	// Vall-e-x
	VallE VallE `yaml:"vall-e"`
}

type VallE struct {
	AudioPath string `yaml:"audio_path"`
}

type FeatureFlag map[string]*bool

func (ff FeatureFlag) Enabled(s string) bool {
	v, exist := ff[s]
	return exist && v != nil && *v
}

type GRPC struct {
	Attempts          int `yaml:"attempts"`
	AttemptsSleepTime int `yaml:"attempts_sleep_time"`
}

type Diffusers struct {
	PipelineType     string  `yaml:"pipeline_type"`
	SchedulerType    string  `yaml:"scheduler_type"`
	CUDA             bool    `yaml:"cuda"`
	EnableParameters string  `yaml:"enable_parameters"` // A list of comma separated parameters to specify
	CFGScale         float32 `yaml:"cfg_scale"`         // Classifier-Free Guidance Scale
	IMG2IMG          bool    `yaml:"img2img"`           // Image to Image Diffuser
	ClipSkip         int     `yaml:"clip_skip"`         // Skip every N frames
	ClipModel        string  `yaml:"clip_model"`        // Clip model to use
	ClipSubFolder    string  `yaml:"clip_subfolder"`    // Subfolder to use for clip model
}

type LLMConfig struct {
	SystemPrompt    string   `yaml:"system_prompt"`
	TensorSplit     string   `yaml:"tensor_split"`
	MainGPU         string   `yaml:"main_gpu"`
	RMSNormEps      float32  `yaml:"rms_norm_eps"`
	NGQA            int32    `yaml:"ngqa"`
	PromptCachePath string   `yaml:"prompt_cache_path"`
	PromptCacheAll  bool     `yaml:"prompt_cache_all"`
	PromptCacheRO   bool     `yaml:"prompt_cache_ro"`
	MirostatETA     float64  `yaml:"mirostat_eta"`
	MirostatTAU     float64  `yaml:"mirostat_tau"`
	Mirostat        int      `yaml:"mirostat"`
	NGPULayers      int      `yaml:"gpu_layers"`
	MMap            bool     `yaml:"mmap"`
	MMlock          bool     `yaml:"mmlock"`
	LowVRAM         bool     `yaml:"low_vram"`
	Grammar         string   `yaml:"grammar"`
	StopWords       []string `yaml:"stopwords"`
	Cutstrings      []string `yaml:"cutstrings"`
	TrimSpace       []string `yaml:"trimspace"`
	ContextSize     int      `yaml:"context_size"`
	NUMA            bool     `yaml:"numa"`
	LoraAdapter     string   `yaml:"lora_adapter"`
	LoraBase        string   `yaml:"lora_base"`
	NoMulMatQ       bool     `yaml:"no_mulmatq"`
	DraftModel      string   `yaml:"draft_model"`
	NDraft          int32    `yaml:"n_draft"`
}

type AutoGPTQ struct {
	ModelBaseName    string `yaml:"model_base_name"`
	Device           string `yaml:"device"`
	Triton           bool   `yaml:"triton"`
	UseFastTokenizer bool   `yaml:"use_fast_tokenizer"`
}

type Functions struct {
	DisableNoAction         bool   `yaml:"disable_no_action"`
	NoActionFunctionName    string `yaml:"no_action_function_name"`
	NoActionDescriptionName string `yaml:"no_action_description_name"`
}

type TemplateConfig struct {
	Chat        string `yaml:"chat"`
	ChatMessage string `yaml:"chat_message"`
	Completion  string `yaml:"completion"`
	Edit        string `yaml:"edit"`
	Functions   string `yaml:"function"`
}

type ConfigLoader struct {
	configs map[string]Config
	sync.Mutex
}

func (c *Config) SetFunctionCallString(s string) {
	c.functionCallString = s
}

func (c *Config) SetFunctionCallNameString(s string) {
	c.functionCallNameString = s
}

func (c *Config) ShouldUseFunctions() bool {
	return ((c.functionCallString != "none" || c.functionCallString == "") || c.ShouldCallSpecificFunction())
}

func (c *Config) ShouldCallSpecificFunction() bool {
	return len(c.functionCallNameString) > 0
}

func (c *Config) FunctionToCall() string {
	return c.functionCallNameString
}

func defaultPredictOptions(modelFile string) PredictionOptions {
	return PredictionOptions{
		TopP:        0.7,
		TopK:        80,
		Maxtokens:   512,
		Temperature: 0.9,
		Model:       modelFile,
	}
}

func DefaultConfig(modelFile string) *Config {
	return &Config{
		PredictionOptions: defaultPredictOptions(modelFile),
	}
}

func NewConfigLoader() *ConfigLoader {
	return &ConfigLoader{
		configs: make(map[string]Config),
	}
}
func ReadConfigFile(file string) ([]*Config, error) {
	c := &[]*Config{}
	f, err := os.ReadFile(file)
	if err != nil {
		return nil, fmt.Errorf("cannot read config file: %w", err)
	}
	if err := yaml.Unmarshal(f, c); err != nil {
		return nil, fmt.Errorf("cannot unmarshal config file: %w", err)
	}

	return *c, nil
}

func ReadConfig(file string) (*Config, error) {
	c := &Config{}
	f, err := os.ReadFile(file)
	if err != nil {
		return nil, fmt.Errorf("cannot read config file: %w", err)
	}
	if err := yaml.Unmarshal(f, c); err != nil {
		return nil, fmt.Errorf("cannot unmarshal config file: %w", err)
	}

	return c, nil
}

func (cm *ConfigLoader) LoadConfigFile(file string) error {
	cm.Lock()
	defer cm.Unlock()
	c, err := ReadConfigFile(file)
	if err != nil {
		return fmt.Errorf("cannot load config file: %w", err)
	}

	for _, cc := range c {
		cm.configs[cc.Name] = *cc
	}
	return nil
}

func (cm *ConfigLoader) LoadConfig(file string) error {
	cm.Lock()
	defer cm.Unlock()
	c, err := ReadConfig(file)
	if err != nil {
		return fmt.Errorf("cannot read config file: %w", err)
	}

	cm.configs[c.Name] = *c
	return nil
}

func (cm *ConfigLoader) GetConfig(m string) (Config, bool) {
	cm.Lock()
	defer cm.Unlock()
	v, exists := cm.configs[m]
	return v, exists
}

func (cm *ConfigLoader) GetAllConfigs() []Config {
	cm.Lock()
	defer cm.Unlock()
	var res []Config
	for _, v := range cm.configs {
		res = append(res, v)
	}
	return res
}

func (cm *ConfigLoader) ListConfigs() []string {
	cm.Lock()
	defer cm.Unlock()
	var res []string
	for k := range cm.configs {
		res = append(res, k)
	}
	return res
}

func (cm *ConfigLoader) LoadConfigs(path string) error {
	cm.Lock()
	defer cm.Unlock()
	entries, err := os.ReadDir(path)
	if err != nil {
		return err
	}
	files := make([]fs.FileInfo, 0, len(entries))
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			return err
		}
		files = append(files, info)
	}
	for _, file := range files {
		// Skip templates, YAML and .keep files
		if !strings.Contains(file.Name(), ".yaml") {
			continue
		}
		c, err := ReadConfig(filepath.Join(path, file.Name()))
		if err == nil {
			cm.configs[c.Name] = *c
		}
	}

	return nil
}
