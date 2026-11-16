package commands

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/lostcoords/wariwari-cli/internal/client"
	"github.com/spf13/cobra"
)

var reminderCmd = &cobra.Command{
	Use:     "reminder",
	Aliases: []string{"reminders"},
	Short:   "Remind members who owe money (Pro)",
}

var reminderParticipant string

const reminderSendHelp = `Remind members of what they owe, by push or email. With --participant,
remind that member; without it, remind everyone who owes and whom you may
remind (an admin, anyone; other members, the people who owe them). Members who
are not reminded are listed under "skipped" with the reason: unlinked,
opted_out or recently_reminded. Each member is reminded at most once a day.
Needs Pro on your account; otherwise 403 PRO_REQUIRED.`

var reminderSendCmd = &cobra.Command{
	Use:   "send [GROUP_TOKEN]",
	Short: "Remind members of what they owe",
	Long:  reminderSendHelp,
	Args:  cobra.MaximumNArgs(1),
	RunE:  runReminderSend,
}

// `remind` is the short form of `reminder send`.
var remindCmd = &cobra.Command{
	Use:   "remind [GROUP_TOKEN]",
	Short: "Remind members of what they owe (same as reminder send)",
	Long:  reminderSendHelp,
	Args:  cobra.MaximumNArgs(1),
	RunE:  runReminderSend,
}

func runReminderSend(cmd *cobra.Command, args []string) error {
	group, err := groupArg(args, 0)
	if err != nil {
		return err
	}
	w := write(nil)
	if reminderParticipant != "" {
		w.Body = map[string]any{"participantToken": reminderParticipant}
	}
	data, err := api.Post(cmd.Context(), "/groups/"+group+"/reminders", w)
	if err != nil {
		return err
	}
	var result struct {
		Reminded []any `json:"reminded"`
		Skipped  []any `json:"skipped"`
	}
	_ = decodeInto(data, &result)
	return printOK(data, fmt.Sprintf("%d reminded, %d skipped", len(result.Reminded), len(result.Skipped)))
}

var reminderScheduleShowCmd = &cobra.Command{
	Use:   "show [GROUP_TOKEN]",
	Short: "Show when reminders go out",
	Long:  `Show the group's reminder schedule, or "reminders": "off" when it has none.`,
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		group, err := groupArg(args, 0)
		if err != nil {
			return err
		}
		data, err := api.Get(cmd.Context(), "/groups/"+group+"/reminder_schedule", nil)
		if absent, err := absentOn404(cmd.Context(), err, groupRecord(group)); absent {
			return printOK(map[string]any{"token": group, "reminders": "off"}, "Reminders off")
		} else if err != nil {
			return err
		}
		return printOK(data, "Reminder schedule")
	},
}

var reminderWeekday string
var reminderDay int

var reminderScheduleCmd = &cobra.Command{
	Use:   "schedule weekly|monthly|off [GROUP_TOKEN]",
	Short: "Turn scheduled reminders on, change them, or turn them off",
	Long: `Remind members who owe money on a schedule. Reminders are off until a group
has one.

  weekly --weekday DAY   every week on DAY (sun..sat, or 0..6 with 0 = Sunday)
  monthly --day N        every month on day N (1..28)
  off                    stop scheduled reminders

"reminder schedule show" prints the current schedule.

Turning them on needs Pro on your account; otherwise 403 PRO_REQUIRED. Turning them off
always works, and succeeds when they are already off.`,
	Args:      cobra.RangeArgs(1, 2),
	ValidArgs: []string{"weekly", "monthly", "off"},
	RunE: func(cmd *cobra.Command, args []string) error {
		group, err := groupArg(args, 1)
		if err != nil {
			return err
		}
		path := "/groups/" + group + "/reminder_schedule"
		switch args[0] {
		case "off":
			if _, err := api.Delete(cmd.Context(), path, client.Write{}); err != nil {
				return err
			}
			return printOK(map[string]any{"token": group, "reminders": "off"}, "Reminders off")
		case "weekly":
			weekday, err := parseWeekday(reminderWeekday)
			if err != nil {
				return err
			}
			data, err := api.Patch(cmd.Context(), path, client.Write{Body: map[string]any{"cadence": "weekly", "weekday": weekday}})
			if err != nil {
				return err
			}
			return printOK(data, "Reminders weekly")
		case "monthly":
			if reminderDay < 1 || reminderDay > 28 {
				return fmt.Errorf("monthly reminders need --day between 1 and 28")
			}
			data, err := api.Patch(cmd.Context(), path, client.Write{Body: map[string]any{"cadence": "monthly", "day": reminderDay}})
			if err != nil {
				return err
			}
			return printOK(data, "Reminders monthly")
		default:
			return fmt.Errorf("unknown schedule %q: use weekly, monthly or off", args[0])
		}
	},
}

var weekdays = []string{"sun", "mon", "tue", "wed", "thu", "fri", "sat"}

func parseWeekday(value string) (int, error) {
	if n, err := strconv.Atoi(value); err == nil && n >= 0 && n <= 6 {
		return n, nil
	}
	name := strings.ToLower(value)
	for i, day := range weekdays {
		if strings.HasPrefix(name, day) {
			return i, nil
		}
	}
	return 0, fmt.Errorf("weekly reminders need --weekday: sun..sat, or 0..6 with 0 = Sunday")
}

func init() {
	rootCmd.AddCommand(reminderCmd, remindCmd)
	reminderCmd.AddCommand(reminderSendCmd, reminderScheduleCmd)
	reminderScheduleCmd.AddCommand(reminderScheduleShowCmd)
	addIdempotencyFlag(reminderSendCmd, remindCmd)
	for _, cmd := range []*cobra.Command{reminderSendCmd, remindCmd} {
		cmd.Flags().StringVar(&reminderParticipant, "participant", "", "Remind only this participant token")
	}
	reminderScheduleCmd.Flags().StringVar(&reminderWeekday, "weekday", "", "Day of the week for weekly reminders (sun..sat or 0..6)")
	reminderScheduleCmd.Flags().IntVar(&reminderDay, "day", 0, "Day of the month for monthly reminders (1..28)")
}
