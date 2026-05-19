package main

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"sort"
	"strings"
	"time"
)

const hostqCronFile = "/etc/cron.d/hostq-user-jobs"

var cronFieldRe = regexp.MustCompile(`^(\*|\d{1,2}|\*/\d{1,2})$`)

func (a *App) cron(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		a.cronAction(w, r)
		return
	}
	a.render(w, "cron", map[string]any{
		"Title":  "Cron Manager",
		"Jobs":   a.listCronJobs(),
		"Output": r.URL.Query().Get("output"),
	})
}

func (a *App) cronAction(w http.ResponseWriter, r *http.Request) {
	action := r.FormValue("action")
	output := ""
	switch action {
	case "create":
		job := CronJob{
			ID:       fmt.Sprintf("%d", time.Now().UnixNano()),
			Name:     cleanCronName(r.FormValue("name")),
			Schedule: cleanCronSchedule(r.FormValue("minute"), r.FormValue("hour"), r.FormValue("day"), r.FormValue("month"), r.FormValue("weekday")),
			User:     cleanCronUser(r.FormValue("user")),
			Command:  cleanCronCommand(r.FormValue("command")),
			Source:   "managed",
		}
		if job.Name == "" || job.Command == "" || job.Schedule == "" {
			output = "Cron job requires a name, schedule, and command."
			break
		}
		if err := a.addCronJob(job); err != nil {
			output = "Cron save failed: " + err.Error()
		} else {
			output = "Cron job added."
			a.audit("cron.create", "success", job.Name)
		}
	case "delete":
		id := strings.TrimSpace(r.FormValue("id"))
		if err := a.deleteCronJob(id); err != nil {
			output = "Cron delete failed: " + err.Error()
		} else {
			output = "Cron job deleted."
			a.audit("cron.delete", "success", id)
		}
	}
	http.Redirect(w, r, "/cron?output="+url.QueryEscape(output), http.StatusSeeOther)
}

func (a *App) listCronJobs() []CronJob {
	jobs := []CronJob{}
	jobs = append(jobs, readHostQCronFile("/etc/cron.d/hostq-backups", "hostQ backups")...)
	jobs = append(jobs, readHostQCronFile(hostqCronFile, "managed")...)
	sort.Slice(jobs, func(i, j int) bool {
		if jobs[i].Source == jobs[j].Source {
			return jobs[i].Name < jobs[j].Name
		}
		return jobs[i].Source < jobs[j].Source
	})
	return jobs
}

func readHostQCronFile(path, source string) []CronJob {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	jobs := []CronJob{}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") || regexp.MustCompile(`^[A-Z_][A-Z0-9_]*=`).MatchString(line) {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 7 {
			continue
		}
		command := strings.Join(fields[6:], " ")
		id := ""
		name := source
		if marker := strings.Index(command, "# hostq:"); marker >= 0 {
			meta := strings.TrimSpace(strings.TrimPrefix(command[marker:], "# hostq:"))
			command = strings.TrimSpace(command[:marker])
			parts := strings.SplitN(meta, ":", 2)
			id = strings.TrimSpace(parts[0])
			if len(parts) > 1 {
				name = strings.TrimSpace(parts[1])
			}
		}
		if id == "" {
			id = source + "-" + strings.ReplaceAll(fields[0]+"-"+fields[1]+"-"+fields[2]+"-"+fields[3]+"-"+fields[4], "*", "x")
		}
		jobs = append(jobs, CronJob{
			ID:       id,
			Name:     name,
			Schedule: strings.Join(fields[0:5], " "),
			User:     fields[5],
			Command:  command,
			Source:   source,
		})
	}
	return jobs
}

func (a *App) addCronJob(job CronJob) error {
	jobs := readHostQCronFile(hostqCronFile, "managed")
	jobs = append(jobs, job)
	return writeManagedCronJobs(jobs)
}

func (a *App) deleteCronJob(id string) error {
	jobs := readHostQCronFile(hostqCronFile, "managed")
	kept := []CronJob{}
	for _, job := range jobs {
		if job.ID != id {
			kept = append(kept, job)
		}
	}
	return writeManagedCronJobs(kept)
}

func writeManagedCronJobs(jobs []CronJob) error {
	var b strings.Builder
	b.WriteString("SHELL=/bin/bash\n")
	b.WriteString("PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin\n")
	b.WriteString("# Managed by hostQ. Edit from the Cron Manager.\n")
	for _, job := range jobs {
		if job.Source != "managed" {
			continue
		}
		b.WriteString(fmt.Sprintf("%s %s %s # hostq:%s:%s\n", job.Schedule, job.User, job.Command, job.ID, job.Name))
	}
	return os.WriteFile(hostqCronFile, []byte(b.String()), 0644)
}

func cleanCronSchedule(minute, hour, day, month, weekday string) string {
	fields := []string{
		cleanCronField(minute, "*"),
		cleanCronField(hour, "*"),
		cleanCronField(day, "*"),
		cleanCronField(month, "*"),
		cleanCronField(weekday, "*"),
	}
	for _, field := range fields {
		if field == "" {
			return ""
		}
	}
	return strings.Join(fields, " ")
}

func cleanCronField(value, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		value = fallback
	}
	if !cronFieldRe.MatchString(value) {
		return ""
	}
	return value
}

func cleanCronName(value string) string {
	value = strings.TrimSpace(value)
	value = regexp.MustCompile(`[^a-zA-Z0-9._ -]+`).ReplaceAllString(value, "-")
	return strings.Trim(value, ". -")
}

func cleanCronUser(value string) string {
	switch strings.TrimSpace(value) {
	case "www-data":
		return "www-data"
	default:
		return "root"
	}
}

func cleanCronCommand(value string) string {
	value = strings.TrimSpace(value)
	value = strings.ReplaceAll(value, "\r", "")
	value = strings.ReplaceAll(value, "\n", "")
	return value
}
