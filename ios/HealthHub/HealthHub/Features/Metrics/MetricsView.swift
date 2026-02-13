import SwiftUI
import Charts

struct MetricsView: View {
    @ObservedObject private var auth = AuthManager.shared
    @State private var showSessionExpired = false
    @State private var showRateLimited = false
    @State private var profiles: [ProfileDTO] = []
    @State private var selectedRange: DateRange = .week
    @State private var dailyMetrics: [DailyMetricData] = []
    @State private var isLoading = false
    @State private var errorMessage: String?
    @State private var selectedDate: Date?

    // Export state
    @State private var isExporting = false
    @State private var exportPhase: String = ""
    @State private var exportError: String?
    @State private var showExportError = false
    @State private var shareFileURL: URL?
    @State private var showShareSheet = false

    // Report history state
    @State private var reports: [ReportDTO] = []
    @State private var isLoadingReports = false
    @State private var reportToDelete: ReportDTO?
    @State private var showDeleteConfirm = false

    enum DateRange: String, CaseIterable {
        case week = "7D"
        case month = "30D"
        case quarter = "90D"

        var days: Int {
            switch self {
            case .week: return 7
            case .month: return 30
            case .quarter: return 90
            }
        }
    }

    private var ownerProfile: ProfileDTO? {
        profiles.first(where: { $0.type == "owner" })
    }

    var body: some View {
        NavigationStack {
            ScrollView {
                VStack(spacing: 16) {
                    // Range selector
                    Picker("Диапазон", selection: $selectedRange) {
                        ForEach(DateRange.allCases, id: \.self) { range in
                            Text(range.rawValue).tag(range)
                        }
                    }
                    .pickerStyle(.segmented)
                    .padding(.horizontal)
                    .onChange(of: selectedRange) { _, _ in
                        Task { await loadMetrics() }
                    }

                    if isLoading {
                        ProgressView("Загрузка...")
                            .padding()
                    } else if let error = errorMessage {
                        VStack(spacing: 8) {
                            Image(systemName: "exclamationmark.triangle")
                                .font(.largeTitle)
                                .foregroundStyle(.orange)
                            Text("Ошибка")
                                .font(.headline)
                            Text(error)
                                .font(.caption)
                                .foregroundStyle(.secondary)
                                .multilineTextAlignment(.center)
                        }
                        .padding()
                    } else {
                        // Charts
                        VStack(spacing: 16) {
                            StepsChartCard(
                                data: dailyMetrics,
                                selectedDate: $selectedDate
                            )

                            WeightChartCard(data: dailyMetrics)

                            RestingHRChartCard(data: dailyMetrics)

                            SleepChartCard(data: dailyMetrics)

                            ActiveEnergyChartCard(data: dailyMetrics)
                        }
                        .padding(.horizontal)
                    }

                    // Export section
                    if ownerProfile != nil {
                        exportSection
                        reportHistorySection
                    }
                }
                .padding(.vertical)
            }
            .navigationTitle("Показатели")
            .refreshable {
                await loadMetrics()
            }
            .task {
                await loadData()
            }
            .sheet(isPresented: $showShareSheet) {
                if let url = shareFileURL {
                    ShareSheet(activityItems: [url])
                }
            }
            .alert("Ошибка", isPresented: $showExportError) {
                Button("OK", role: .cancel) {}
            } message: {
                Text(exportError ?? "Неизвестная ошибка")
            }
            .alert("Удалить отчёт?", isPresented: $showDeleteConfirm) {
                Button("Удалить", role: .destructive) {
                    if let report = reportToDelete {
                        Task { await performDeleteReport(report) }
                    }
                }
                Button("Отмена", role: .cancel) {}
            } message: {
                if let report = reportToDelete {
                    let formatStr: String = report.format.uppercased()
                    let fromStr: String = report.from
                    let toStr: String = report.to
                    Text(formatStr + " " + fromStr + " — " + toStr)
                }
            }
            .alert("Сессия истекла", isPresented: $showSessionExpired) {
                Button("OK") {
                    auth.handleUnauthorized()
                }
            } message: {
                Text("Войдите заново")
            }
            .alert("Слишком много запросов", isPresented: $showRateLimited) {
                Button("OK", role: .cancel) {}
            } message: {
                Text("Попробуйте позже")
            }
        }
    }

    private func handleError(_ error: Error) -> Bool {
        if let apiError = error as? APIError, apiError == .unauthorized {
            showSessionExpired = true
            return true
        }
        if let apiError = error as? APIError, apiError == .rateLimited {
            showRateLimited = true
            return true
        }
        return false
    }

    // MARK: - Export Section

    @ViewBuilder
    private var exportSection: some View {
        VStack(alignment: .leading, spacing: 12) {
            Text("Экспорт")
                .font(.headline)

            if isExporting {
                HStack(spacing: 8) {
                    ProgressView()
                    Text(exportPhase)
                        .font(.subheadline)
                        .foregroundStyle(.secondary)
                }
                .frame(maxWidth: .infinity, alignment: .center)
                .padding(.vertical, 4)
            } else {
                HStack(spacing: 12) {
                    Button {
                        Task { await exportReport(format: "pdf") }
                    } label: {
                        Label("Экспорт PDF", systemImage: "doc.richtext")
                            .frame(maxWidth: .infinity)
                    }
                    .buttonStyle(.bordered)
                    .tint(.red)

                    Button {
                        Task { await exportReport(format: "csv") }
                    } label: {
                        Label("Экспорт CSV", systemImage: "tablecells")
                            .frame(maxWidth: .infinity)
                    }
                    .buttonStyle(.bordered)
                    .tint(.green)
                }
            }
        }
        .padding()
        .background(
            RoundedRectangle(cornerRadius: 12)
                .fill(Color(.systemBackground))
                .shadow(color: .black.opacity(0.05), radius: 4, x: 0, y: 2)
        )
        .padding(.horizontal)
    }

    // MARK: - Report History Section

    @ViewBuilder
    private var reportHistorySection: some View {
        VStack(alignment: .leading, spacing: 12) {
            HStack {
                Text("История отчётов")
                    .font(.headline)
                Spacer()
                if isLoadingReports {
                    ProgressView()
                        .controlSize(.small)
                }
            }

            if reports.isEmpty && !isLoadingReports {
                Text("Нет отчётов")
                    .font(.subheadline)
                    .foregroundStyle(.secondary)
                    .frame(maxWidth: .infinity, alignment: .center)
                    .padding(.vertical, 8)
            } else {
                LazyVStack(spacing: 0) {
                    ForEach(reports) { report in
                        reportRow(report)
                        if report.id != reports.last?.id {
                            Divider()
                        }
                    }
                }
            }
        }
        .padding()
        .background(
            RoundedRectangle(cornerRadius: 12)
                .fill(Color(.systemBackground))
                .shadow(color: .black.opacity(0.05), radius: 4, x: 0, y: 2)
        )
        .padding(.horizontal)
    }

    private func reportRow(_ report: ReportDTO) -> some View {
        let fromDate: String = report.from
        let toDate: String = report.to
        let dateRangeText = fromDate + " — " + toDate

        return HStack {
            VStack(alignment: .leading, spacing: 4) {
                HStack(spacing: 6) {
                    Text(report.format.uppercased())
                        .font(.caption2)
                        .fontWeight(.bold)
                        .padding(.horizontal, 6)
                        .padding(.vertical, 2)
                        .background(report.format == "pdf" ? Color.red.opacity(0.1) : Color.green.opacity(0.1))
                        .foregroundStyle(report.format == "pdf" ? .red : .green)
                        .clipShape(RoundedRectangle(cornerRadius: 4))

                    Text(dateRangeText)
                        .font(.subheadline)
                }

                HStack(spacing: 8) {
                    Text(formatDate(report.createdAt))
                        .font(.caption)
                        .foregroundStyle(.secondary)

                    if report.sizeBytes > 0 {
                        Text(formatBytes(report.sizeBytes))
                            .font(.caption)
                            .foregroundStyle(.secondary)
                    }
                }
            }

            Spacer()

            Button {
                Task { await downloadAndShare(report) }
            } label: {
                Image(systemName: "square.and.arrow.down")
                    .font(.body)
            }
            .buttonStyle(.borderless)
            .disabled(isExporting)

            Button {
                reportToDelete = report
                showDeleteConfirm = true
            } label: {
                Image(systemName: "trash")
                    .font(.body)
                    .foregroundStyle(.red)
            }
            .buttonStyle(.borderless)
        }
        .padding(.vertical, 6)
    }

    // MARK: - Data Loading

    private func apiDay(_ date: Date) -> String {
        let f = DateFormatter()
        f.calendar = Calendar(identifier: .gregorian)
        f.locale = Locale(identifier: "en_US_POSIX")
        f.timeZone = TimeZone.current
        f.dateFormat = "yyyy-MM-dd"
        return f.string(from: date)
    }

    private func loadData() async {
        do {
            profiles = try await APIClient.shared.listProfiles()
            await loadMetrics()
            await loadReports()
        } catch {
            if !handleError(error) {
                errorMessage = "Не удалось загрузить профили: \(error.localizedDescription)"
            }
        }
    }

    private func loadMetrics() async {
        guard let ownerProfile = profiles.first(where: { $0.type == "owner" }) else {
            errorMessage = "Owner profile not found"
            return
        }

        isLoading = true
        errorMessage = nil

        let toDate = Date()
        let fromDate = Calendar.current.date(byAdding: .day, value: -selectedRange.days, to: toDate)!

        do {
            let items = try await APIClient.shared.fetchDailyMetrics(
                profileId: ownerProfile.id,
                from: apiDay(fromDate),
                to: apiDay(toDate)
            )

            // Parse response
            dailyMetrics = items.compactMap { item in
                parseDailyMetric(item)
            }
        } catch {
            if !handleError(error) {
                errorMessage = "Не удалось загрузить метрики: \(error.localizedDescription)"
            }
        }

        isLoading = false
    }

    private func loadReports() async {
        guard let profile = ownerProfile else { return }
        isLoadingReports = true
        do {
            reports = try await APIClient.shared.listReports(profileId: profile.id, limit: 10)
        } catch {
            // Silently fail — history is non-critical
        }
        isLoadingReports = false
    }

    // MARK: - Export Actions

    private func exportReport(format: String) async {
        guard let profile = ownerProfile else {
            exportError = "Профиль не найден"
            showExportError = true
            return
        }

        isExporting = true
        exportPhase = "Готовим отчёт…"

        let to = dateString(from: Date())
        let from = dateString(from: Calendar.current.date(
            byAdding: .day,
            value: -(selectedRange.days - 1),
            to: Date()
        )!)

        do {
            let report = try await APIClient.shared.createReport(
                profileId: profile.id,
                from: from,
                to: to,
                format: format
            )

            exportPhase = "Скачиваем…"
            let fileURL = try await APIClient.shared.downloadReport(report: report)

            isExporting = false
            shareFileURL = fileURL
            showShareSheet = true

            await loadReports()
        } catch let error as APIError {
            isExporting = false
            switch error {
            case .serverError(let code):
                if code == "invalid_range" || code == "range_too_large" {
                    exportError = "Слишком большой период. Максимум 90 дней."
                } else {
                    exportError = "Ошибка сервера: \(code)"
                }
            default:
                exportError = "Не удалось создать отчёт."
            }
            showExportError = true
        } catch {
            isExporting = false
            exportError = "Не удалось создать отчёт: \(error.localizedDescription)"
            showExportError = true
        }
    }

    private func downloadAndShare(_ report: ReportDTO) async {
        isExporting = true
        exportPhase = "Скачиваем…"

        do {
            let fileURL = try await APIClient.shared.downloadReport(report: report)
            isExporting = false
            shareFileURL = fileURL
            showShareSheet = true
        } catch {
            isExporting = false
            exportError = "Не удалось скачать отчёт: \(error.localizedDescription)"
            showExportError = true
        }
    }

    private func performDeleteReport(_ report: ReportDTO) async {
        do {
            try await APIClient.shared.deleteReport(reportId: report.id)
            await loadReports()
        } catch {
            exportError = "Не удалось удалить отчёт: \(error.localizedDescription)"
            showExportError = true
        }
    }

    // MARK: - Parsing & Formatting

    private func parseDailyMetric(_ item: DailyAggregate) -> DailyMetricData? {
        guard let date = parseDate(item.date) else { return nil }

        var steps: Int?
        var weight: Double?
        var restingHR: Int?
        var sleepMinutes: Int?
        var activeEnergy: Int?

        // Parse activity
        if let activity = item.activity {
            steps = activity.steps
            if let ae = activity.activeEnergyKcal, ae > 0 {
                activeEnergy = ae
            }
        }

        // Parse body
        if let body = item.body {
            if let w = body.weightKgLast, w > 0 {
                weight = w
            }
        }

        // Parse heart
        if let heart = item.heart {
            if let hr = heart.restingHrBpm, hr > 0 {
                restingHR = hr
            }
        }

        // Parse sleep
        if let sleep = item.sleep {
            if let total = sleep.totalMinutes, total > 0 {
                sleepMinutes = total
            }
        }

        return DailyMetricData(
            date: date,
            steps: steps,
            weight: weight,
            restingHR: restingHR,
            sleepMinutes: sleepMinutes,
            activeEnergyKcal: activeEnergy
        )
    }

    private func parseDate(_ dateString: String) -> Date? {
        let formatter = ISO8601DateFormatter()
        formatter.formatOptions = [.withFullDate]
        return formatter.date(from: dateString)
    }

    private func dateString(from date: Date) -> String {
        let formatter = DateFormatter()
        formatter.dateFormat = "yyyy-MM-dd"
        formatter.locale = Locale(identifier: "en_US_POSIX")
        return formatter.string(from: date)
    }

    private func formatDate(_ date: Date) -> String {
        let formatter = DateFormatter()
        formatter.dateStyle = .short
        formatter.timeStyle = .short
        formatter.locale = Locale(identifier: "ru_RU")
        return formatter.string(from: date)
    }

    private func formatBytes(_ bytes: Int64) -> String {
        if bytes < 1024 {
            return "\(bytes) Б"
        } else if bytes < 1024 * 1024 {
            return String(format: "%.1f КБ", Double(bytes) / 1024.0)
        } else {
            return String(format: "%.1f МБ", Double(bytes) / (1024.0 * 1024.0))
        }
    }
}

// MARK: - Data Model

struct DailyMetricData: Identifiable {
    let id = UUID()
    let date: Date
    let steps: Int?
    let weight: Double?
    let restingHR: Int?
    let sleepMinutes: Int?
    let activeEnergyKcal: Int?
}

// MARK: - Steps Chart Card

struct StepsChartCard: View {
    let data: [DailyMetricData]
    @Binding var selectedDate: Date?

    private var stepsData: [(Date, Int)] {
        data.compactMap { item in
            guard let steps = item.steps else { return nil }
            return (item.date, steps)
        }
    }

    var body: some View {
        VStack(alignment: .leading, spacing: 12) {
            HStack {
                Image(systemName: "figure.walk")
                    .foregroundStyle(.blue)
                Text("Шаги")
                    .font(.headline)
                Spacer()
                if let selected = selectedDate,
                   let value = stepsData.first(where: { Calendar.current.isDate($0.0, inSameDayAs: selected) })?.1 {
                    Text("\(value)")
                        .font(.caption)
                        .foregroundStyle(.secondary)
                }
            }

            if stepsData.isEmpty {
                Text("Нет данных")
                    .font(.caption)
                    .foregroundStyle(.secondary)
                    .frame(height: 150)
                    .frame(maxWidth: .infinity, alignment: .center)
            } else {
                Chart {
                    ForEach(stepsData, id: \.0) { date, steps in
                        AreaMark(
                            x: .value("Дата", date),
                            y: .value("Шаги", steps)
                        )
                        .foregroundStyle(
                            .linearGradient(
                                colors: [.blue.opacity(0.5), .blue.opacity(0.1)],
                                startPoint: .top,
                                endPoint: .bottom
                            )
                        )

                        LineMark(
                            x: .value("Дата", date),
                            y: .value("Шаги", steps)
                        )
                        .foregroundStyle(.blue)
                        .lineStyle(StrokeStyle(lineWidth: 2))
                    }

                    if let selected = selectedDate {
                        RuleMark(x: .value("Selected", selected))
                            .foregroundStyle(.gray.opacity(0.5))
                            .lineStyle(StrokeStyle(lineWidth: 1, dash: [5]))
                    }
                }
                .frame(height: 150)
                .chartXAxis {
                    AxisMarks(values: .stride(by: .day, count: max(1, stepsData.count / 5))) { _ in
                        AxisValueLabel(format: .dateTime.day().month(.abbreviated))
                    }
                }
                .chartYAxis {
                    AxisMarks(position: .leading)
                }
                .chartOverlay { proxy in
                    GeometryReader { geometry in
                        Rectangle()
                            .fill(Color.clear)
                            .contentShape(Rectangle())
                            .gesture(
                                DragGesture(minimumDistance: 0)
                                    .onChanged { value in
                                        let x = value.location.x
                                        if let date: Date = proxy.value(atX: x) {
                                            selectedDate = date
                                        }
                                    }
                                    .onEnded { _ in
                                        selectedDate = nil
                                    }
                            )
                    }
                }
            }
        }
        .padding()
        .background(
            RoundedRectangle(cornerRadius: 12)
                .fill(Color(.systemBackground))
                .shadow(color: .black.opacity(0.05), radius: 4, x: 0, y: 2)
        )
    }
}

// MARK: - Weight Chart Card

struct WeightChartCard: View {
    let data: [DailyMetricData]

    private var weightData: [(Date, Double)] {
        data.compactMap { item in
            guard let weight = item.weight else { return nil }
            return (item.date, weight)
        }
    }

    var body: some View {
        VStack(alignment: .leading, spacing: 12) {
            HStack {
                Image(systemName: "scalemass")
                    .foregroundStyle(.green)
                Text("Вес")
                    .font(.headline)
                Spacer()
                if let latest = weightData.last?.1 {
                    Text(String(format: "%.1f кг", latest))
                        .font(.caption)
                        .foregroundStyle(.secondary)
                }
            }

            if weightData.isEmpty {
                Text("Нет данных")
                    .font(.caption)
                    .foregroundStyle(.secondary)
                    .frame(height: 150)
                    .frame(maxWidth: .infinity, alignment: .center)
            } else {
                Chart {
                    ForEach(weightData, id: \.0) { date, weight in
                        LineMark(
                            x: .value("Дата", date),
                            y: .value("Вес", weight)
                        )
                        .foregroundStyle(.green)
                        .lineStyle(StrokeStyle(lineWidth: 2))
                        .symbol(Circle())
                    }
                }
                .frame(height: 150)
                .chartXAxis {
                    AxisMarks(values: .stride(by: .day, count: max(1, weightData.count / 5))) { _ in
                        AxisValueLabel(format: .dateTime.day().month(.abbreviated))
                    }
                }
                .chartYAxis {
                    AxisMarks(position: .leading)
                }
            }
        }
        .padding()
        .background(
            RoundedRectangle(cornerRadius: 12)
                .fill(Color(.systemBackground))
                .shadow(color: .black.opacity(0.05), radius: 4, x: 0, y: 2)
        )
    }
}

// MARK: - Resting HR Chart Card

struct RestingHRChartCard: View {
    let data: [DailyMetricData]

    private var hrData: [(Date, Int)] {
        data.compactMap { item in
            guard let hr = item.restingHR else { return nil }
            return (item.date, hr)
        }
    }

    var body: some View {
        VStack(alignment: .leading, spacing: 12) {
            HStack {
                Image(systemName: "heart.fill")
                    .foregroundStyle(.red)
                Text("Пульс покоя")
                    .font(.headline)
                Spacer()
                if let latest = hrData.last?.1 {
                    Text("\(latest) bpm")
                        .font(.caption)
                        .foregroundStyle(.secondary)
                }
            }

            if hrData.isEmpty {
                Text("Нет данных")
                    .font(.caption)
                    .foregroundStyle(.secondary)
                    .frame(height: 150)
                    .frame(maxWidth: .infinity, alignment: .center)
            } else {
                Chart {
                    ForEach(hrData, id: \.0) { date, hr in
                        LineMark(
                            x: .value("Дата", date),
                            y: .value("Пульс", hr)
                        )
                        .foregroundStyle(.red)
                        .lineStyle(StrokeStyle(lineWidth: 2))
                        .symbol(Circle())
                    }
                }
                .frame(height: 150)
                .chartXAxis {
                    AxisMarks(values: .stride(by: .day, count: max(1, hrData.count / 5))) { _ in
                        AxisValueLabel(format: .dateTime.day().month(.abbreviated))
                    }
                }
                .chartYAxis {
                    AxisMarks(position: .leading)
                }
            }
        }
        .padding()
        .background(
            RoundedRectangle(cornerRadius: 12)
                .fill(Color(.systemBackground))
                .shadow(color: .black.opacity(0.05), radius: 4, x: 0, y: 2)
        )
    }
}

// MARK: - Sleep Chart Card

struct SleepChartCard: View {
    let data: [DailyMetricData]

    private var sleepData: [(Date, Int)] {
        data.compactMap { item in
            guard let sleep = item.sleepMinutes else { return nil }
            return (item.date, sleep)
        }
    }

    var body: some View {
        VStack(alignment: .leading, spacing: 12) {
            HStack {
                Image(systemName: "bed.double.fill")
                    .foregroundStyle(.purple)
                Text("Сон")
                    .font(.headline)
                Spacer()
                if let latest = sleepData.last?.1 {
                    let hours = latest / 60
                    let minutes = latest % 60
                    Text("\(hours)ч \(minutes)м")
                        .font(.caption)
                        .foregroundStyle(.secondary)
                }
            }

            if sleepData.isEmpty {
                Text("Нет данных")
                    .font(.caption)
                    .foregroundStyle(.secondary)
                    .frame(height: 150)
                    .frame(maxWidth: .infinity, alignment: .center)
            } else {
                Chart {
                    ForEach(sleepData, id: \.0) { date, minutes in
                        BarMark(
                            x: .value("Дата", date),
                            y: .value("Минуты", minutes)
                        )
                        .foregroundStyle(.purple.gradient)
                    }
                }
                .frame(height: 150)
                .chartXAxis {
                    AxisMarks(values: .stride(by: .day, count: max(1, sleepData.count / 5))) { _ in
                        AxisValueLabel(format: .dateTime.day().month(.abbreviated))
                    }
                }
                .chartYAxis {
                    AxisMarks(position: .leading) { value in
                        if let minutes = value.as(Int.self) {
                            let hours = minutes / 60
                            AxisValueLabel {
                                Text("\(hours)ч")
                            }
                        }
                    }
                }
            }
        }
        .padding()
        .background(
            RoundedRectangle(cornerRadius: 12)
                .fill(Color(.systemBackground))
                .shadow(color: .black.opacity(0.05), radius: 4, x: 0, y: 2)
        )
    }
}

// MARK: - Active Energy Chart Card

struct ActiveEnergyChartCard: View {
    let data: [DailyMetricData]

    private var energyData: [(Date, Int)] {
        data.compactMap { item in
            guard let energy = item.activeEnergyKcal else { return nil }
            return (item.date, energy)
        }
    }

    var body: some View {
        VStack(alignment: .leading, spacing: 12) {
            HStack {
                Image(systemName: "flame.fill")
                    .foregroundStyle(.orange)
                Text("Активная энергия")
                    .font(.headline)
                Spacer()
                if let latest = energyData.last?.1 {
                    Text("\(latest) ккал")
                        .font(.caption)
                        .foregroundStyle(.secondary)
                }
            }

            if energyData.isEmpty {
                Text("Нет данных")
                    .font(.caption)
                    .foregroundStyle(.secondary)
                    .frame(height: 150)
                    .frame(maxWidth: .infinity, alignment: .center)
            } else {
                Chart {
                    ForEach(energyData, id: \.0) { date, kcal in
                        AreaMark(
                            x: .value("Дата", date),
                            y: .value("ккал", kcal)
                        )
                        .foregroundStyle(
                            .linearGradient(
                                colors: [.orange.opacity(0.5), .orange.opacity(0.1)],
                                startPoint: .top,
                                endPoint: .bottom
                            )
                        )

                        LineMark(
                            x: .value("Дата", date),
                            y: .value("ккал", kcal)
                        )
                        .foregroundStyle(.orange)
                        .lineStyle(StrokeStyle(lineWidth: 2))
                        .symbol(Circle())
                    }
                }
                .frame(height: 150)
                .chartXAxis {
                    AxisMarks(values: .stride(by: .day, count: max(1, energyData.count / 5))) { _ in
                        AxisValueLabel(format: .dateTime.day().month(.abbreviated))
                    }
                }
                .chartYAxis {
                    AxisMarks(position: .leading)
                }
            }
        }
        .padding()
        .background(
            RoundedRectangle(cornerRadius: 12)
                .fill(Color(.systemBackground))
                .shadow(color: .black.opacity(0.05), radius: 4, x: 0, y: 2)
        )
    }
}

struct MetricsView_Previews: PreviewProvider {
    static var previews: some View {
        MetricsView()
    }
}
