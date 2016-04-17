package etc

import (
	"io/ioutil"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/mkideal/cli"
	"github.com/mkideal/pkg/debug"
	yaml "gopkg.in/yaml.v2"
)

type Config struct {
	Host           string `yaml:"host" cli:"H,host" usage:"listening host" dft:"0.0.0.0"`
	Port           uint16 `yaml:"port" cli:"p,port" usage:"listening port" dft:"25"`
	Debug          bool   `yaml:"debug" cli:"debug" usage:"enable debug mode" dft:"false"`
	DBSource       string `yaml:"db_source" cli:"db-source" usage:"mysql db source" dft:"$SMTPD_DB_SOURCE"`
	DomainName     string `yaml:"domain_name" cli:"dn,domain-name" usage:"my domain name"`
	MaxSessionSize int    `yaml:"max_session_size" cli:"max-session-size" usage:"max size of sessions(less than max size of open files)" dft:"32768"`
	MaxErrorSize   int    `yaml:"max_error_size" cli:"max-error-size" usage:"max size of errors" dft:"3"`
	MaxBufferSize  int    `yaml:"max_buffer_size" cli:"max-buffer-szie" usage:"max size of buffer" dft:"6553600"`
	MaxRecipients  int    `yaml:"max_recipients" cli:"max-recipients" usage:"max size of recipients" dft:"256"`
	AllowDelay     bool   `yaml:"allow_delay" cli:"allow-delay" usage:"allow delay email" dft:"false"`

	S_ServiceInfo string `yaml:"service_info" cli:"-"`
}

//-------------
// Load config
//-------------

type configMeta struct {
	locker      sync.Mutex
	filename    string
	modTime     int64
	args        []string
	initialized int32

	conf Config
}

var meta = &configMeta{}

// LoadConfig loads yaml config. load from defaultConfigFile if configFile==""
func LoadConfig(filename string, replaceArgs []string) error {
	meta.locker.Lock()
	defer meta.locker.Unlock()

	if filename != "" {
		meta.filename = filename
	}
	meta.args = replaceArgs

	file, err := os.Open(meta.filename)
	if err != nil {
		return err
	}
	if finfo, err := file.Stat(); err == nil {
		newModTime := finfo.ModTime().Unix()
		if newModTime == atomic.LoadInt64(&meta.modTime) {
			return nil
		}
		atomic.StoreInt64(&meta.modTime, newModTime)
	}

	newConf := Config{}
	if data, err := ioutil.ReadAll(file); err != nil {
		return err
	} else if err = yaml.Unmarshal(data, &newConf); err != nil {
		return err
	}
	if replaceArgs != nil {
		cli.Parse(replaceArgs, &newConf)
	}
	atomic.StoreInt32(&meta.initialized, 1)

	meta.conf = newConf
	debug.Switch(meta.conf.Debug)
	debug.Debugf("config:\n%v", debug.JSON(meta.conf))
	return nil
}

func Conf() Config {
	return meta.conf
}

func init() {
	go func() {
		for {
			<-time.After(time.Second * 10)
			if atomic.LoadInt32(&meta.initialized) != 0 {
				LoadConfig(meta.filename, meta.args)
			}
		}
	}()
}
