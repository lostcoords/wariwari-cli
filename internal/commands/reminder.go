package commands

import (
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/lostcoords/wariwari-cli/internal/client"
	"github.com/spf13/cobra"
)

var reminderCmd = &cobra.Command{
	Use:     "reminder",
	Aliases: []string{"reminders"},
	Short:   "Remind members who owe money (Pro groups)",
}

var reminderParticipant string

const reminderSendHelp = `Send a reminder of what they owe. With --participant, remind that member;
without it, remind every member the snapshot shows owing money in any
currency. A member reminded recently is skipped (RATE_LIMIT_EXCEEDED) and
listed under "skipped". Needs a Pro group; otherwise 403 PRO_REQUIRED.`

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
	path := "/groups/" + group + "/reminders"
	if reminderParticipant != "" {
		data, err := api.Post(cmd.Context(), path, write(map[string]any{"participantToken": reminderParticipant}))
		if err != nil {
			return err
		}
		return printOK(data, "Reminder sent")
	}
	if idempotencyKeyFlag != "" {
		return fmt.Errorf("--idempotency-key needs --participant: one key covers one reminder")
	}

	snapshot, err := groupSnapshot(cmd.Context(), group)
	if err != nil {
		return err
	}
	reminders := []any{}
	skipped := []map[string]any{}
	for _, participant := range owingParticipants(snapshot) {
		data, err := api.Post(cmd.Context(), path, write(map[string]any{"participantToken": participant}))
		var apiErr *client.Error
		switch {
		case errors.As(err, &apiErr) && apiErr.Code() == "RATE_LIMIT_EXCEEDED":
			skipped = append(skipped, map[string]any{"participantToken": participant, "code": apiErr.Code(), "detail": apiErr.Detail()})
		case err != nil:
			return err
		default:
			reminders = append(reminders, data)
		}
	}
	return printOK(map[string]any{"reminders": reminders, "skipped": skipped},
		fmt.Sprintf("%d reminded, %d skipped", len(reminders), len(skipped)))
}

// owingParticipants reads netByCurrency, where a negative balance in any
// currency means the member owes money.
func owingParticipants(snapshot map[string]any) []string {
	owing := map[string]bool{}
	buckets, _ := snapshot["netByCurrency"].(map[string]any)
	for _, bucket := range buckets {
		balances, _ := bucket.(map[string]any)
		for token, cents := range balances {
			if value, _ := cents.(float64); value < 0 {
				owing[token] = true
			}
		}
	}
	tokens := make([]string, 0, len(owing))
	for token := range owing {
		tokens = append(tokens, token)
	}
	sort.Strings(tokens)
	return tokens
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

Turning them on needs a Pro group; otherwise 403 PRO_REQUIRED. Turning them off
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
	addIdempotencyFlag(reminderSendCmd, remindCmd)
	for _, cmd := range []*cobra.Command{reminderSendCmd, remindCmd} {
		cmd.Flags().StringVar(&reminderParticipant, "participant", "", "Remind only this participant token")
	}
	reminderScheduleCmd.Flags().StringVar(&reminderWeekday, "weekday", "", "Day of the week for weekly reminders (sun..sat or 0..6)")
	reminderScheduleCmd.Flags().IntVar(&reminderDay, "day", 0, "Day of the month for monthly reminders (1..28)")
}
