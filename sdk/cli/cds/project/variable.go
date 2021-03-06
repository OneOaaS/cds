package project

import (
	"fmt"
	"gopkg.in/yaml.v2"

	"github.com/spf13/cobra"

	"github.com/ovh/cds/sdk"
)

// CmdVariable Command to manage variable on project
var CmdVariable = &cobra.Command{
	Use:     "variable",
	Short:   "",
	Long:    ``,
	Aliases: []string{"v"},
}

func init() {
	CmdVariable.AddCommand(cmdProjectShowVariable())
	CmdVariable.AddCommand(cmdProjectAddVariable())
	CmdVariable.AddCommand(cmdProjectUpdateVariable())
	CmdVariable.AddCommand(cmdProjectRemoveVariable())
}

func cmdProjectShowVariable() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show",
		Short: "cds project variable show <projectKey>",
		Long:  ``,
		Run:   showVarInProject,
	}
	return cmd
}

func showVarInProject(cmd *cobra.Command, args []string) {
	if len(args) != 1 {
		sdk.Exit("Wrong usage: %s\n", cmd.Short)
	}
	projectKey := args[0]

	variables, err := sdk.ShowVariableInProject(projectKey)
	if err != nil {
		sdk.Exit("Error: cannot show variables for project %s (%s)\n", projectKey, err)
	}

	data, err := yaml.Marshal(variables)
	if err != nil {
		sdk.Exit("Error: cannot format output (%s)\n", err)
	}

	fmt.Println(string(data))
}

func cmdProjectAddVariable() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add",
		Short: "cds project variable add <projectKey> <variableName> <variableValue> <variableType>",
		Long:  ``,
		Run:   addVarInProject,
	}
	return cmd
}

func addVarInProject(cmd *cobra.Command, args []string) {
	if len(args) != 4 {
		sdk.Exit("Wrong usage: %s\n", cmd.Short)
	}
	projectKey := args[0]
	varName := args[1]
	varValue := args[2]
	varType := args[3]

	err := sdk.AddVariableInProject(projectKey, varName, varValue, varType)
	if err != nil {
		sdk.Exit("Error: cannot add variable %s in project %s (%s)\n", varName, projectKey, err)
	}
	fmt.Printf("OK\n")
}

func cmdProjectUpdateVariable() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update",
		Short: "cds project variable update <projectKey> <variableName> <variableValue> <variableType>",
		Long:  ``,
		Run:   updateVarInProject,
	}
	return cmd
}

func updateVarInProject(cmd *cobra.Command, args []string) {
	if len(args) != 4 {
		sdk.Exit("Wrong usage: %s\n", cmd.Short)
	}
	projectKey := args[0]
	varName := args[1]
	varValue := args[2]
	varType := args[3]

	err := sdk.UpdateVariableInProject(projectKey, varName, varValue, varType)
	if err != nil {
		sdk.Exit("Error: cannot update variable %s in project %s (%s)\n", varName, projectKey, err)
	}
	fmt.Printf("OK\n")
}

func cmdProjectRemoveVariable() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "remove",
		Short: "cds project variable remove <projectKey> <variableName>",
		Long:  ``,
		Run:   removeVarFromProject,
	}
	return cmd
}

func removeVarFromProject(cmd *cobra.Command, args []string) {
	if len(args) != 2 {
		sdk.Exit("Wrong usage: %s\n", cmd.Short)
	}
	projectKey := args[0]
	varName := args[1]

	err := sdk.RemoveVariableFromProject(projectKey, varName)
	if err != nil {
		sdk.Exit("Error: cannot remove variable %s from project %s (%s)\n", varName, projectKey, err)
	}
	fmt.Printf("OK\n")
}
