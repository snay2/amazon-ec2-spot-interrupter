// Copyright Amazon.com Inc. or its affiliates. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License"). You may
// not use this file except in compliance with the License. A copy of the
// License is located at
//
//     http://aws.amazon.com/apache2.0/
//
// or in the "license" file accompanying this file. This file is distributed
// on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either
// express or implied. See the License for the specific language governing
// permissions and limitations under the License.

package main

import (
	"context"
	"fmt"
	"time"

	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/itn/pkg/itn"
	tea "github.com/charmbracelet/bubbletea"
)

type model struct {
	choices     []ec2types.Instance
	cursor      int
	selected    map[int]*ec2types.Instance
	ctx         context.Context
	itn         *itn.ITN
	initialized bool
}

type spotInstancesMsg []ec2types.Instance

func NewModel(ctx context.Context, itn *itn.ITN) model {
	return model{
		selected: map[int]*ec2types.Instance{},
		ctx:      ctx,
		itn:      itn,
	}
}

func initialModel(ctx context.Context, itn *itn.ITN) tea.Cmd {
	return func() tea.Msg {
		instances, err := itn.SpotInstances(ctx)
		if err != nil {
			panic(err)
		}
		return spotInstancesMsg(instances)
	}
}

func (m model) Init() tea.Cmd {
	return initialModel(m.ctx, m.itn)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case spotInstancesMsg:
		m.choices = msg
		m.initialized = true
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.choices)-1 {
				m.cursor++
			}
		case " ":
			if _, ok := m.selected[m.cursor]; ok {
				delete(m.selected, m.cursor)
			} else {
				m.selected[m.cursor] = &m.choices[m.cursor]
			}
		case "enter":
			var instanceIDs []string
			for _, instance := range m.selected {
				instanceIDs = append(instanceIDs, *instance.InstanceId)
			}
			if err := m.itn.Interrupt(m.ctx, instanceIDs, time.Second*15, true); err != nil {
				return m, tea.Quit
			}
			return m, tea.Quit
		}
	}
	return m, nil
}

func instanceName(i ec2types.Instance) string {
	for _, tag := range i.Tags {
		if *tag.Key == "Name" {
			return *tag.Value
		}
	}
	return ""
}

func (m model) View() string {
	if !m.initialized {
		return "Finding Spot instances...\n\n\nPress q to quit.\n"
	}
	if len(m.choices) == 0 {
		return "There are currently no Spot instances running...\n\n\nPress q to quit.\n"
	}
	// The header
	s := "Which Spot instances would you like to interrupt?\n\n"

	// Iterate over our choices
	for i, choice := range m.choices {

		// Is the cursor pointing at this choice?
		cursor := " " // no cursor
		if m.cursor == i {
			cursor = ">" // cursor!
		}

		// Is this choice selected?
		checked := " " // not selected
		if _, ok := m.selected[i]; ok {
			checked = "x" // selected!
		}

		// Render the row
		s += fmt.Sprintf("%s [%s] %s (%s)\n", cursor, checked, *choice.InstanceId, instanceName(choice))
	}

	// The footer
	s += "\nPress q to quit.\n"

	// Send the UI for rendering
	return s
}
