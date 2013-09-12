package cmd

import (
	"bufio"
	"errors"
	"github.com/alexzorin/onapp"
	"github.com/alexzorin/onapp/cmd/log"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

const (
	vmCmdDescription     = "Manage virtual machines"
	vmCmdHelp            = "See subcommands for help on managing virtual machines."
	vmCmdListDescription = "List virtual machines under your account"
	vmCmdListHelp        = "\nUsage: `onapp vm list [filter]`\n" +
		"Optionally filter by field query, e.gg onapp vm list [Label=prod Hostname=.com User=1 Memory=1024]. (case sensitive)"
	vmCmdStartDescription        = "Boots a virtual machine"
	vmCmdStartHelp               = "Boots virtual machine by id: `onapp vm start <id>."
	vmCmdStopDescription         = "Stops a virtual machine"
	vmCmdStopHelp                = "Stops a virtual machine by id: `onapp vm stop <id>`."
	vmCmdRebootDescription       = "Reboots a virtual machine"
	vmCmdRebootHelp              = "Reboots a virtual machine by id: `onapp vm stop <id>`."
	vmCmdTransactionsDescription = "Lists recent transactions on a virtual machine"
	vmCmdTransactionsHelp        = "Usage: `onapp vm transactions <id> [number_to_list]`"
)

// Base command

type vmCmd struct{}

var vmCmdHandlers = map[string]cmdHandler{
	"list":   vmCmdList{},
	"start":  vmCmdStart{},
	"stop":   vmCmdStop{},
	"reboot": vmCmdReboot{},
	"tx":     vmCmdTransactions{},
}

func (c vmCmd) Run(args []string, ctx *cli) error {
	if len(args) == 0 {
		log.Infoln("This command does nothing when invoked on its own.")
		cmdHandlers["help"].Run([]string{"vm"}, ctx)
		return nil
	} else {
		return ctx.subhandle(c, args)
	}
}

func (c vmCmd) Description() string {
	return vmCmdDescription
}

func (c vmCmd) Help(args []string) {
	log.Infoln(vmCmdHelp)
}

func (c vmCmd) Handlers() *map[string]cmdHandler {
	return &vmCmdHandlers
}

// List command
type vmCmdList struct{}

func (c vmCmdList) Run(args []string, ctx *cli) error {
	list, err := ctx.apiClient.GetVirtualMachines()
	if err != nil {
		return err
	}
	sort.Sort(list)
	var searches []search
	pattern := regexp.MustCompile("^(\\w+)=(\\w+)$")
	for _, s := range args {
		matches := pattern.FindStringSubmatch(strings.Trim(s, " "))
		if len(matches) != 3 {
			log.Warnf("Search query '%s' isn't valid\n", s)
		} else {
			searches = append(searches, search{matches[1], matches[2]})
		}
	}
	asList := list.AsList()
	for _, s := range searches {
		asList = ctx.Search(s, asList)
	}
	for item := asList.Front(); item != nil; item = item.Next() {
		vm := (item.Value).(onapp.VirtualMachine)
		log.Infof("%25.25s   #%-3d   HV-%-2d   User %-4d   %-18s %2d CPUs  %6dM RAM   %15s   %-30.25s\n",
			vm.Label, vm.Id, vm.HV, vm.User, vm.BootedStringColored(), vm.Cpus, vm.Memory, vm.GetIpAddress().Address, vm.Template)
	}
	return nil
}

func (c vmCmdList) Description() string {
	return vmCmdListDescription
}

func (c vmCmdList) Help(args []string) {
	log.Infoln(vmCmdListHelp)
	log.Infoln("\nField names are as follows: ")
	log.Infof("%+v\n\n", &onapp.VirtualMachine{})
}

// Start command
type vmCmdStart struct{}

func (c vmCmdStart) Run(args []string, ctx *cli) error {
	if len(args) == 0 {
		c.Help(args)
		return nil
	} else {
		id, err := strconv.Atoi(strings.Trim(args[0], " "))
		if err != nil {
			return err
		}
		busy := ctx.checkVmBusy(id)
		if busy != nil {
			return busy
		}
		return ctx.apiClient.VirtualMachineStartup(id)
	}
}

func (c vmCmdStart) Description() string {
	return vmCmdStartDescription
}

func (c vmCmdStart) Help(args []string) {
	log.Infoln(vmCmdStartHelp)
}

// Stop command
type vmCmdStop struct{}

func (c vmCmdStop) Run(args []string, ctx *cli) error {
	if len(args) == 0 {
		c.Help(args)
		return nil
	} else {
		id, err := strconv.Atoi(strings.Trim(args[0], " "))
		if err != nil {
			return err
		}
		busy := ctx.checkVmBusy(id)
		if busy != nil {
			return busy
		}
		return ctx.apiClient.VirtualMachineShutdown(id)
	}
}

func (c vmCmdStop) Description() string {
	return vmCmdStopDescription
}

func (c vmCmdStop) Help(args []string) {
	log.Infoln(vmCmdStopHelp)
}

// Reboot command
type vmCmdReboot struct{}

func (c vmCmdReboot) Run(args []string, ctx *cli) error {
	if len(args) == 0 {
		c.Help(args)
		return nil
	} else {
		id, err := strconv.Atoi(strings.Trim(args[0], " "))
		if err != nil {
			return err
		}
		busy := ctx.checkVmBusy(id)
		if busy != nil {
			return busy
		}
		return ctx.apiClient.VirtualMachineReboot(id)
	}
}

func (c vmCmdReboot) Description() string {
	return vmCmdRebootDescription
}

func (c vmCmdReboot) Help(args []string) {
	log.Infoln(vmCmdRebootHelp)
}

// Transactions command
type vmCmdTransactions struct{}

func (c vmCmdTransactions) Run(args []string, ctx *cli) error {
	if len(args) == 0 {
		c.Help(args)
		return nil
	}
	id, err := strconv.Atoi(strings.Trim(args[0], " "))
	if err != nil {
		return nil
	}
	nList := 10
	if len(args) == 2 {
		nList, err = strconv.Atoi(strings.Trim(args[1], " "))
		if err != nil {
			return err
		}
	}
	txns, err := ctx.apiClient.VirtualMachineGetTransactions(id)
	if err != nil {
		return err
	}
	for i := 0; i <= nList && i < len(txns); i++ {
		tx := txns[i]
		t, err := tx.CreatedAtTime()
		if err != nil {
			log.Errorln(err)
			continue
		}
		log.Infof("%25.25s   #%-6d   %-25.25s   %10s\n", t, tx.Id, tx.Action, tx.StatusColored())
	}
	return nil
}

func (c vmCmdTransactions) Description() string {
	return vmCmdTransactionsDescription
}

func (c vmCmdTransactions) Help(args []string) {
	log.Infoln(vmCmdTransactionsHelp)
}

// Shared funcs

func (ctx *cli) checkVmBusy(id int) error {
	busy, err := ctx.apiClient.VirtualMachineGetRunningTransaction(id)
	if err != nil {
		return err
	}
	if busy.IsValid() {
		log.Warnf("This VM is currently running a transaction: %s\n", busy.Action)
		log.Warnf("Do you want to queue another action anyway? [y/n]: ")

		reader := bufio.NewReader(os.Stdin)
		resp, err := reader.ReadString('\n')
		if err != nil {
			return err
		}
		if strings.ToLower(resp)[0] == 'n' {
			return errors.New("User cancelled action")
		}
	}
	return nil
}
