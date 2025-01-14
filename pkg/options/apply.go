package options

import (
	"bufio"
	"bytes"
	"errors"
	"io"
	"os"

	"kcl-lang.io/krm-kcl/pkg/kio"
	"kcl-lang.io/kubectl-kcl/pkg/client"
	"kcl-lang.io/kcl-go/pkg/logger"
)

// ApplyOptions is the options for the apply sub command.
type ApplyOptions struct {
	// InputPath is the -f flag
	InputPath string
	// OutputPath is the -o flag
	OutputPath string

	// Namespace is the -n flag --namespace. It will set kubernetes namespace scope
	Namespace string

	// Selector is the -l flag --selector. It will return by label selector
	Selector string

	// FieldSelector is the --field-selector. Selector (field query) to filter on, supports '=', '==', and '!='.(e.g. --field-selector
	// key1=value1,key2=value2). The server only supports a limited number of field queries per type
	FieldSelector string
}

// NewRunOptions() create a new apply options for the apply command.
func NewApplyOptions() *ApplyOptions {
	return &ApplyOptions{}
}

// Run apply command option.
func (o *ApplyOptions) Run() error {
	var buf bytes.Buffer
	cli := o.getCliRuntime()
	if err := cli.GetGeneralResources(&buf); err != nil {
		logger.GetLogger().Errorf("get general resource err: %s", err.Error())
		return err
	}
	_, bs, err := o.reader()
	if err != nil {
		logger.GetLogger().Errorf("read kcl code err: %s", err.Error())
		return err
	}
	n := append(buf.Bytes(), bs...)
	reader := bytes.NewReader(n)

	// use the io pipe feat to connect io writer to io reader
	pr, pw := io.Pipe()
	go func() {
		pipeline := kio.NewPipeline(reader, pw, true)
		if err := pipeline.Execute(); err != nil {
			logger.GetLogger().Errorf("pipeline execute err: %s", err.Error())
		}
		defer pw.Close()
	}()
	var input bytes.Buffer
	_, err = input.ReadFrom(pr)
	if err != nil {
		logger.GetLogger().Errorf("io reader err: %s", err.Error())
		return err
	}
	if err := cli.Apply(cli.Flags, &input); err != nil {
		logger.GetLogger().Errorf("cli apply err: %s", err.Error())
		return err
	}
	return nil
}

func (o *ApplyOptions) getCliRuntime() *client.KubeCliRuntime {
	cliRuntime := client.NewKubeCliRuntime()
	cliRuntime.Namespace = o.Namespace
	cliRuntime.Selector = o.Selector
	cliRuntime.FieldSelector = o.FieldSelector
	return cliRuntime
}

// Validate the options.
func (o *ApplyOptions) Validate() error {
	return nil
}

func (o *ApplyOptions) reader() (io.Reader, []byte, error) {
	if o.InputPath == "-" {
		scanner := bufio.NewScanner(os.Stdin)
		if !scanner.Scan() {
			return nil, nil, errors.New("input scan err")
		}
		line := scanner.Bytes()
		return os.Stdin, line, nil
	} else {
		file, err := os.Open(o.InputPath)
		if err != nil {
			return nil, nil, err
		}
		stat, err := file.Stat()
		if err != nil {
			return nil, nil, err
		}
		bs := make([]byte, stat.Size())
		_, err = bufio.NewReader(file).Read(bs)
		if err != nil {
			return nil, nil, err
		}
		return bufio.NewReader(file), bs, nil
	}
}

func (o *ApplyOptions) writer() (io.Writer, error) {
	if o.OutputPath == "" {
		return os.Stdout, nil
	} else {
		file, err := os.OpenFile(o.OutputPath, os.O_CREATE|os.O_RDWR, 0744)
		if err != nil {
			return nil, err
		}
		return bufio.NewWriter(file), nil
	}
}
