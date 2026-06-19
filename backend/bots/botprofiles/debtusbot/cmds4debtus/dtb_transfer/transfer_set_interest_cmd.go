package dtb_transfer

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/bots-go-framework/bots-fw/botinput"
	"github.com/bots-go-framework/bots-fw/botmsg"
	"github.com/bots-go-framework/bots-fw/botsfw"
	"github.com/sneat-co/debtus/backend/debtus/models4debtus"
	"github.com/sneat-co/sneat-translations/trans"

	"github.com/crediterra/go-interest"
	"github.com/strongo/decimal"
)

var reInterest = regexp.MustCompile(`^\s*(?P<percent>\d+(?:[\.,]\d+)?)%?(?:/(?P<period>\d+|w(?:eek)?|y(?:ear)?|m(?:onth)?))?(?:/(?P<minimum>\d+))?(?:/(?P<grace>\d+))?(?::\s*(?P<comment>.+?))?\s*$`)

func interestAction(whc botsfw.WebhookContext, nextAction botsfw.CommandAction) (m botmsg.MessageFromBot, err error) {
	mt := whc.Input().(botinput.TextMessage).Text()

	if matches := reInterest.FindStringSubmatch(mt); len(matches) > 0 {
		chatEntity := whc.ChatData()

		var data models4debtus.TransferInterest

		for i, name := range reInterest.SubexpNames() {
			v := matches[i]
			switch name {
			case "percent":
				v = strings.Replace(v, ",", ".", 1)
				if data.InterestPercent, err = decimal.ParseDecimal64p2(v); err != nil {
					return
				}
			case "period":
				if v == "" {
					m.Text = whc.Translate(trans.MESSAGE_TEXT_INTEREST_PLEASE_SPECIFY_PERIOD)
					return
				}
				switch v[0] {
				case "d"[0]:
					data.InterestPeriod = interest.RatePeriodDaily
				case "w"[0]:
					data.InterestPeriod = interest.RatePeriodWeekly
				case "m"[0]:
					data.InterestPeriod = interest.RatePeriodMonthly
				case "y"[0]:
					data.InterestPeriod = interest.RatePeriodYearly
				default:
					var vInt int
					if vInt, err = strconv.Atoi(v); err != nil {
						return
					}
					data.InterestPeriod = interest.RatePeriodInDays(vInt)
				}
			case "minimum":
				if v != "" {
					if data.InterestMinimumPeriod, err = strconv.Atoi(v); err != nil {
						return
					}
				}
			case "grace":
				if v != "" {
					if data.InterestGracePeriod, err = strconv.Atoi(v); err != nil {
						return
					}
				}
			case "comment":
				chatEntity.AddWizardParam(transferWizardParamComment, v)
			}
		}
		chatEntity.AddWizardParam(TRANSFER_WIZARD_PARAM_INTEREST, fmt.Sprintf("%v/%v/%v/%v/%v",
			interest.FormulaSimple, data.InterestPercent, data.InterestPeriod, data.InterestMinimumPeriod, data.InterestGracePeriod),
		)

		return nextAction(whc)
	}

	return
}

const TRANSFER_WIZARD_PARAM_INTEREST = "interest"

func getInterestData(s string) (transferInterest models4debtus.TransferInterest, err error) {
	v := strings.Split(s, "/")
	formula := interest.Formula(v[0])
	if !interest.IsKnownFormula(formula) {
		return transferInterest, fmt.Errorf("unknown interest formula=%v", formula)
	}
	var (
		period  int
		percent decimal.Decimal64p2
	)
	if period, err = strconv.Atoi(v[2]); err != nil {
		return transferInterest, err
	}
	if percent, err = decimal.ParseDecimal64p2(v[1]); err != nil {
		return transferInterest, err
	} else {
		transferInterest = models4debtus.NewInterest(formula, percent, interest.RatePeriodInDays(period))
	}

	if minimumPeriod, err := strconv.Atoi(v[3]); err != nil {
		return transferInterest, err
	} else {
		transferInterest = transferInterest.WithMinimumPeriod(minimumPeriod)
	}

	if gracePeriod, err := strconv.Atoi(v[4]); err != nil {
		return transferInterest, err
	} else {
		transferInterest = transferInterest.WithGracePeriod(gracePeriod)
	}

	return transferInterest, nil
}
