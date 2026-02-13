import Foundation
import HealthKit

class HealthKitManager {
    static let shared = HealthKitManager()

    private let healthStore = HKHealthStore()
    private var observerQueries: [HKObserverQuery] = []

    private init() {}

    // MARK: - Authorization

    var isHealthDataAvailable: Bool {
        return HKHealthStore.isHealthDataAvailable()
    }

    private func backgroundSampleTypes() -> [HKSampleType] {
        var types: [HKSampleType] = [
            HKObjectType.quantityType(forIdentifier: .stepCount)!,
            HKObjectType.quantityType(forIdentifier: .heartRate)!,
            HKObjectType.quantityType(forIdentifier: .restingHeartRate)!,
            HKObjectType.quantityType(forIdentifier: .bodyMass)!,
            HKObjectType.categoryType(forIdentifier: .sleepAnalysis)!,
            HKObjectType.quantityType(forIdentifier: .activeEnergyBurned)!,
            HKObjectType.quantityType(forIdentifier: .dietaryEnergyConsumed)!,
            HKObjectType.quantityType(forIdentifier: .dietaryProtein)!,
            HKObjectType.quantityType(forIdentifier: .dietaryFatTotal)!,
            HKObjectType.quantityType(forIdentifier: .dietaryCarbohydrates)!,
            HKObjectType.quantityType(forIdentifier: .dietaryCalcium)!,
            HKObjectType.workoutType()
        ]

        if let wristTemp = HKObjectType.quantityType(forIdentifier: .appleSleepingWristTemperature) {
            types.append(wristTemp)
        }

        return types
    }

    private func deliveryFrequency(for type: HKSampleType) -> HKUpdateFrequency {
        if let quantityType = type as? HKQuantityType {
            switch HKQuantityTypeIdentifier(rawValue: quantityType.identifier) {
            case .stepCount,
                 .heartRate,
                 .restingHeartRate,
                 .activeEnergyBurned,
                 .dietaryEnergyConsumed,
                 .dietaryProtein,
                 .dietaryFatTotal,
                 .dietaryCarbohydrates,
                 .dietaryCalcium:
                return .hourly
            default:
                return .immediate
            }
        }
        return .immediate
    }

    func requestAuthorization() async throws {
        guard isHealthDataAvailable else {
            throw HealthKitError.notAvailable
        }

        var typesToRead: Set<HKObjectType> = [
            HKObjectType.quantityType(forIdentifier: .stepCount)!,
            HKObjectType.quantityType(forIdentifier: .heartRate)!,
            HKObjectType.quantityType(forIdentifier: .restingHeartRate)!,
            HKObjectType.quantityType(forIdentifier: .bodyMass)!,

            // Sleep
            HKObjectType.categoryType(forIdentifier: .sleepAnalysis)!,

            // Workouts
            HKObjectType.workoutType(),

            // Nutrition
            HKObjectType.quantityType(forIdentifier: .dietaryEnergyConsumed)!,
            HKObjectType.quantityType(forIdentifier: .dietaryProtein)!,
            HKObjectType.quantityType(forIdentifier: .dietaryFatTotal)!,
            HKObjectType.quantityType(forIdentifier: .dietaryCarbohydrates)!,
            HKObjectType.quantityType(forIdentifier: .dietaryCalcium)!,

            // Activity
            HKObjectType.quantityType(forIdentifier: .activeEnergyBurned)!,
            HKObjectType.quantityType(forIdentifier: .distanceWalkingRunning)!,
            HKObjectType.quantityType(forIdentifier: .appleExerciseTime)!,
            HKObjectType.categoryType(forIdentifier: .appleStandHour)!
        ]

        // Wrist temperature (iOS 16+, может быть недоступно на некоторых устройствах)
        if let wristTemp = HKObjectType.quantityType(forIdentifier: .appleSleepingWristTemperature) {
            typesToRead.insert(wristTemp)
        }

        // Types to write (for intakes)
        var typesToWrite: Set<HKSampleType> = []

        if let waterType = HKObjectType.quantityType(forIdentifier: .dietaryWater) {
            typesToWrite.insert(waterType)
        }

        // Add dietary types for supplements (if needed)
        // Note: Only standard HealthKit types can be written
        // Most supplement components can be mapped to dietary types

        try await healthStore.requestAuthorization(toShare: typesToWrite, read: typesToRead)
    }

    // MARK: - Background Delivery

    func enableBackgroundDelivery() async throws {
        for type in backgroundSampleTypes() {
            try await withCheckedThrowingContinuation { (continuation: CheckedContinuation<Void, Error>) in
                healthStore.enableBackgroundDelivery(for: type, frequency: deliveryFrequency(for: type)) { success, error in
                    if let error = error {
                        continuation.resume(throwing: error)
                        return
                    }

                    if success {
                        continuation.resume(returning: ())
                    } else {
                        continuation.resume(throwing: HealthKitError.authorizationFailed)
                    }
                }
            }
        }
    }

    func startObserverQueries(onUpdate: @escaping () -> Void) {
        stopObserverQueries()

        for type in backgroundSampleTypes() {
            let query = HKObserverQuery(sampleType: type, predicate: nil) { _, completionHandler, error in
                defer { completionHandler() }

                if let error = error {
                    #if DEBUG
                    print("HealthKit observer error (\(type.identifier)): \(error)")
                    #endif
                    return
                }

                DispatchQueue.main.async {
                    onUpdate()
                }
            }

            observerQueries.append(query)
            healthStore.execute(query)
        }
    }

    func stopObserverQueries() {
        for query in observerQueries {
            healthStore.stop(query)
        }
        observerQueries.removeAll()
    }

    // MARK: - Write Intakes to HealthKit

    /// Write water intake to HealthKit
    func writeWater(amountMl: Int, date: Date = Date()) async throws {
        guard let waterType = HKQuantityType.quantityType(forIdentifier: .dietaryWater) else {
            throw HealthKitError.typeNotAvailable
        }

        let quantity = HKQuantity(unit: HKUnit.literUnit(with: .milli), doubleValue: Double(amountMl))
        let sample = HKQuantitySample(type: waterType, quantity: quantity, start: date, end: date)

        try await healthStore.save(sample)
    }

    /// Write supplement components to HealthKit (if supported)
    /// Only writes components that have valid HealthKit identifiers
    func writeSupplementComponents(_ components: [SupplementComponentDTO], date: Date = Date()) async {
        for component in components {
            guard let hkId = component.hkIdentifier else { continue }

            do {
                try await writeDietaryComponent(hkIdentifier: hkId, amount: component.amount, unit: component.unit, date: date)
            } catch {
                print("Failed to write component \(component.nutrientKey): \(error)")
                // Continue with other components
            }
        }
    }

    private func writeDietaryComponent(hkIdentifier: String, amount: Double, unit: String, date: Date) async throws {
        // Map HealthKit identifier string to HKQuantityTypeIdentifier
        guard let identifier = mapToHKIdentifier(hkIdentifier) else {
            throw HealthKitError.typeNotAvailable
        }

        guard let quantityType = HKQuantityType.quantityType(forIdentifier: identifier) else {
            throw HealthKitError.typeNotAvailable
        }

        // Map unit string to HKUnit
        let hkUnit = mapToHKUnit(unit)

        let quantity = HKQuantity(unit: hkUnit, doubleValue: amount)
        let sample = HKQuantitySample(type: quantityType, quantity: quantity, start: date, end: date)

        try await healthStore.save(sample)
    }

    private func mapToHKIdentifier(_ hkString: String) -> HKQuantityTypeIdentifier? {
        // Map common supplement identifiers
        switch hkString {
        case "dietaryVitaminA": return .dietaryVitaminA
        case "dietaryVitaminB6": return .dietaryVitaminB6
        case "dietaryVitaminB12": return .dietaryVitaminB12
        case "dietaryVitaminC": return .dietaryVitaminC
        case "dietaryVitaminD": return .dietaryVitaminD
        case "dietaryVitaminE": return .dietaryVitaminE
        case "dietaryVitaminK": return .dietaryVitaminK
        case "dietaryCalcium": return .dietaryCalcium
        case "dietaryIron": return .dietaryIron
        case "dietaryMagnesium": return .dietaryMagnesium
        case "dietaryZinc": return .dietaryZinc
        case "dietaryPotassium": return .dietaryPotassium
        default: return nil
        }
    }

    private func mapToHKUnit(_ unitString: String) -> HKUnit {
        switch unitString.lowercased() {
        case "mg": return HKUnit.gramUnit(with: .milli)
        case "mcg", "μg": return HKUnit.gramUnit(with: .micro)
        case "g": return HKUnit.gram()
        case "ml": return HKUnit.literUnit(with: .milli)
        case "l": return HKUnit.liter()
        case "iu": return HKUnit.internationalUnit()
        default: return HKUnit.gramUnit(with: .milli) // Default to mg
        }
    }

    // MARK: - Daily Steps

    func fetchDailySteps(from: Date, to: Date) async throws -> [(date: String, steps: Int)] {
        let stepType = HKQuantityType.quantityType(forIdentifier: .stepCount)!

        let calendar = Calendar.current
        let startOfDay = calendar.startOfDay(for: from)
        let endOfDay = calendar.date(byAdding: .day, value: 1, to: calendar.startOfDay(for: to))!

        let predicate = HKQuery.predicateForSamples(withStart: startOfDay, end: endOfDay, options: .strictStartDate)

        var anchorComponents = calendar.dateComponents([.year, .month, .day], from: startOfDay)
        anchorComponents.hour = 0
        let anchorDate = calendar.date(from: anchorComponents)!

        let interval = DateComponents(day: 1)

        return try await withCheckedThrowingContinuation { continuation in
            let query = HKStatisticsCollectionQuery(
                quantityType: stepType,
                quantitySamplePredicate: predicate,
                options: .cumulativeSum,
                anchorDate: anchorDate,
                intervalComponents: interval
            )

            query.initialResultsHandler = { query, results, error in
                if let error = error {
                    continuation.resume(throwing: error)
                    return
                }

                guard let results = results else {
                    continuation.resume(returning: [])
                    return
                }

                var dailySteps: [(date: String, steps: Int)] = []
                let formatter = DateFormatter()
                formatter.dateFormat = "yyyy-MM-dd"

                results.enumerateStatistics(from: startOfDay, to: endOfDay) { statistics, stop in
                    if let sum = statistics.sumQuantity() {
                        let steps = Int(sum.doubleValue(for: HKUnit.count()))
                        let dateString = formatter.string(from: statistics.startDate)
                        dailySteps.append((date: dateString, steps: steps))
                    }
                }

                continuation.resume(returning: dailySteps)
            }

            healthStore.execute(query)
        }
    }

    // MARK: - Hourly Steps

    func fetchHourlySteps(for date: Date) async throws -> [HourlyBucket] {
        let stepType = HKQuantityType.quantityType(forIdentifier: .stepCount)!

        let calendar = Calendar.current
        let startOfDay = calendar.startOfDay(for: date)
        let endOfDay = calendar.date(byAdding: .day, value: 1, to: startOfDay)!

        let predicate = HKQuery.predicateForSamples(withStart: startOfDay, end: endOfDay, options: .strictStartDate)

        var anchorComponents = calendar.dateComponents([.year, .month, .day], from: startOfDay)
        anchorComponents.hour = 0
        let anchorDate = calendar.date(from: anchorComponents)!

        let interval = DateComponents(hour: 1)

        return try await withCheckedThrowingContinuation { continuation in
            let query = HKStatisticsCollectionQuery(
                quantityType: stepType,
                quantitySamplePredicate: predicate,
                options: .cumulativeSum,
                anchorDate: anchorDate,
                intervalComponents: interval
            )

            query.initialResultsHandler = { query, results, error in
                if let error = error {
                    continuation.resume(throwing: error)
                    return
                }

                guard let results = results else {
                    continuation.resume(returning: [])
                    return
                }

                var hourlyBuckets: [HourlyBucket] = []

                results.enumerateStatistics(from: startOfDay, to: endOfDay) { statistics, stop in
                    if let sum = statistics.sumQuantity() {
                        let steps = Int(sum.doubleValue(for: HKUnit.count()))
                        hourlyBuckets.append(HourlyBucket(
                            hour: statistics.startDate,
                            steps: steps,
                            hr: nil
                        ))
                    }
                }

                continuation.resume(returning: hourlyBuckets)
            }

            healthStore.execute(query)
        }
    }

    // MARK: - Hourly Heart Rate

    func fetchHourlyHeartRate(for date: Date) async throws -> [HourlyBucket] {
        let hrType = HKQuantityType.quantityType(forIdentifier: .heartRate)!

        let calendar = Calendar.current
        let startOfDay = calendar.startOfDay(for: date)
        let endOfDay = calendar.date(byAdding: .day, value: 1, to: startOfDay)!

        let predicate = HKQuery.predicateForSamples(withStart: startOfDay, end: endOfDay, options: .strictStartDate)

        var anchorComponents = calendar.dateComponents([.year, .month, .day], from: startOfDay)
        anchorComponents.hour = 0
        let anchorDate = calendar.date(from: anchorComponents)!

        let interval = DateComponents(hour: 1)

        return try await withCheckedThrowingContinuation { continuation in
            let query = HKStatisticsCollectionQuery(
                quantityType: hrType,
                quantitySamplePredicate: predicate,
                options: [.discreteMin, .discreteMax, .discreteAverage],
                anchorDate: anchorDate,
                intervalComponents: interval
            )

            query.initialResultsHandler = { query, results, error in
                if let error = error {
                    continuation.resume(throwing: error)
                    return
                }

                guard let results = results else {
                    continuation.resume(returning: [])
                    return
                }

                var hourlyBuckets: [HourlyBucket] = []

                results.enumerateStatistics(from: startOfDay, to: endOfDay) { statistics, stop in
                    if let min = statistics.minimumQuantity(),
                       let max = statistics.maximumQuantity(),
                       let avg = statistics.averageQuantity() {

                        let minBpm = Int(min.doubleValue(for: HKUnit.count().unitDivided(by: .minute())))
                        let maxBpm = Int(max.doubleValue(for: HKUnit.count().unitDivided(by: .minute())))
                        let avgBpm = Int(avg.doubleValue(for: HKUnit.count().unitDivided(by: .minute())))

                        hourlyBuckets.append(HourlyBucket(
                            hour: statistics.startDate,
                            steps: nil,
                            hr: HRData(min: minBpm, max: maxBpm, avg: avgBpm)
                        ))
                    }
                }

                continuation.resume(returning: hourlyBuckets)
            }

            healthStore.execute(query)
        }
    }

    // MARK: - Daily Resting HR

    func fetchDailyRestingHR(from: Date, to: Date) async throws -> [String: Int] {
        let restingHRType = HKQuantityType.quantityType(forIdentifier: .restingHeartRate)!

        let calendar = Calendar.current
        let startOfDay = calendar.startOfDay(for: from)
        let endOfDay = calendar.date(byAdding: .day, value: 1, to: calendar.startOfDay(for: to))!

        let predicate = HKQuery.predicateForSamples(withStart: startOfDay, end: endOfDay, options: .strictStartDate)

        var anchorComponents = calendar.dateComponents([.year, .month, .day], from: startOfDay)
        anchorComponents.hour = 0
        let anchorDate = calendar.date(from: anchorComponents)!

        let interval = DateComponents(day: 1)

        return try await withCheckedThrowingContinuation { continuation in
            let query = HKStatisticsCollectionQuery(
                quantityType: restingHRType,
                quantitySamplePredicate: predicate,
                options: .discreteAverage,
                anchorDate: anchorDate,
                intervalComponents: interval
            )

            query.initialResultsHandler = { query, results, error in
                if let error = error {
                    continuation.resume(throwing: error)
                    return
                }

                guard let results = results else {
                    continuation.resume(returning: [:])
                    return
                }

                var dailyRestingHR: [String: Int] = [:]
                let formatter = DateFormatter()
                formatter.dateFormat = "yyyy-MM-dd"

                results.enumerateStatistics(from: startOfDay, to: endOfDay) { statistics, stop in
                    if let avg = statistics.averageQuantity() {
                        let bpm = Int(avg.doubleValue(for: HKUnit.count().unitDivided(by: .minute())))
                        let dateString = formatter.string(from: statistics.startDate)
                        dailyRestingHR[dateString] = bpm
                    }
                }

                continuation.resume(returning: dailyRestingHR)
            }

            healthStore.execute(query)
        }
    }

    // MARK: - Daily Nutrition

    func fetchDailyNutrition(from: Date, to: Date) async throws -> [String: NutritionDaily] {
        let calendar = Calendar.current
        let startOfDay = calendar.startOfDay(for: from)
        let endOfDay = calendar.date(byAdding: .day, value: 1, to: calendar.startOfDay(for: to))!
        let formatter = DateFormatter()
        formatter.dateFormat = "yyyy-MM-dd"

        // Fetch all nutrition types concurrently
        async let energyData = fetchDailySum(.dietaryEnergyConsumed, from: startOfDay, to: endOfDay, unit: .kilocalorie())
        async let proteinData = fetchDailySum(.dietaryProtein, from: startOfDay, to: endOfDay, unit: .gram())
        async let fatData = fetchDailySum(.dietaryFatTotal, from: startOfDay, to: endOfDay, unit: .gram())
        async let carbsData = fetchDailySum(.dietaryCarbohydrates, from: startOfDay, to: endOfDay, unit: .gram())
        async let calciumData = fetchDailySum(.dietaryCalcium, from: startOfDay, to: endOfDay, unit: .gramUnit(with: .milli))

        let energy = try await energyData
        let protein = try await proteinData
        let fat = try await fatData
        let carbs = try await carbsData
        let calcium = try await calciumData

        // Combine by date
        var result: [String: NutritionDaily] = [:]
        let allDates = Set(energy.keys).union(protein.keys).union(fat.keys).union(carbs.keys).union(calcium.keys)

        for date in allDates {
            let e = Int(energy[date] ?? 0)
            let p = Int(protein[date] ?? 0)
            let f = Int(fat[date] ?? 0)
            let c = Int(carbs[date] ?? 0)
            let ca = Int(calcium[date] ?? 0)

            // Only include if at least one value > 0
            if e > 0 || p > 0 || f > 0 || c > 0 || ca > 0 {
                result[date] = NutritionDaily(
                    energyKcal: e,
                    proteinG: p,
                    fatG: f,
                    carbsG: c,
                    calciumMg: ca
                )
            }
        }

        return result
    }

    private func fetchDailySum(_ identifier: HKQuantityTypeIdentifier, from: Date, to: Date, unit: HKUnit) async throws -> [String: Double] {
        guard let type = HKQuantityType.quantityType(forIdentifier: identifier) else {
            return [:]
        }

        let calendar = Calendar.current
        let predicate = HKQuery.predicateForSamples(withStart: from, end: to, options: .strictStartDate)

        var anchorComponents = calendar.dateComponents([.year, .month, .day], from: from)
        anchorComponents.hour = 0
        let anchorDate = calendar.date(from: anchorComponents)!

        let interval = DateComponents(day: 1)

        return try await withCheckedThrowingContinuation { continuation in
            let query = HKStatisticsCollectionQuery(
                quantityType: type,
                quantitySamplePredicate: predicate,
                options: .cumulativeSum,
                anchorDate: anchorDate,
                intervalComponents: interval
            )

            query.initialResultsHandler = { query, results, error in
                if let error = error {
                    continuation.resume(throwing: error)
                    return
                }

                guard let results = results else {
                    continuation.resume(returning: [:])
                    return
                }

                var dailyData: [String: Double] = [:]
                let formatter = DateFormatter()
                formatter.dateFormat = "yyyy-MM-dd"

                results.enumerateStatistics(from: from, to: to) { statistics, stop in
                    if let sum = statistics.sumQuantity() {
                        let value = sum.doubleValue(for: unit)
                        if value > 0 {
                            let dateString = formatter.string(from: statistics.startDate)
                            dailyData[dateString] = value
                        }
                    }
                }

                continuation.resume(returning: dailyData)
            }

            healthStore.execute(query)
        }
    }

    // MARK: - Daily Activity

    func fetchDailyActivity(from: Date, to: Date) async throws -> [String: (activeEnergy: Int, distance: Double, exerciseMin: Int, standHours: Int)] {
        let calendar = Calendar.current
        let startOfDay = calendar.startOfDay(for: from)
        let endOfDay = calendar.date(byAdding: .day, value: 1, to: calendar.startOfDay(for: to))!

        async let energyData = fetchDailySum(.activeEnergyBurned, from: startOfDay, to: endOfDay, unit: .kilocalorie())
        async let distanceData = fetchDailySum(.distanceWalkingRunning, from: startOfDay, to: endOfDay, unit: .meter())
        async let exerciseData = fetchDailySum(.appleExerciseTime, from: startOfDay, to: endOfDay, unit: .minute())
        async let standData = fetchDailyStandHours(from: startOfDay, to: endOfDay)

        let energy = try await energyData
        let distance = try await distanceData
        let exercise = try await exerciseData
        let stand = try await standData

        var result: [String: (activeEnergy: Int, distance: Double, exerciseMin: Int, standHours: Int)] = [:]
        let allDates = Set(energy.keys).union(distance.keys).union(exercise.keys).union(stand.keys)

        for date in allDates {
            let e = Int(energy[date] ?? 0)
            let d = (distance[date] ?? 0) / 1000.0 // meters to km
            let ex = Int(exercise[date] ?? 0)
            let s = stand[date] ?? 0

            if e > 0 || d > 0 || ex > 0 || s > 0 {
                result[date] = (activeEnergy: e, distance: d, exerciseMin: ex, standHours: s)
            }
        }

        return result
    }

    private func fetchDailyStandHours(from: Date, to: Date) async throws -> [String: Int] {
        guard let standType = HKObjectType.categoryType(forIdentifier: .appleStandHour) else {
            return [:]
        }

        let calendar = Calendar.current
        let formatter = DateFormatter()
        formatter.dateFormat = "yyyy-MM-dd"

        var dailyStand: [String: Int] = [:]
        var currentDate = calendar.startOfDay(for: from)
        let endDate = calendar.date(byAdding: .day, value: 1, to: calendar.startOfDay(for: to))!

        while currentDate < endDate {
            let nextDate = calendar.date(byAdding: .day, value: 1, to: currentDate)!
            let predicate = HKQuery.predicateForSamples(withStart: currentDate, end: nextDate, options: .strictStartDate)

            let count = try await withCheckedThrowingContinuation { (continuation: CheckedContinuation<Int, Error>) in
                let query = HKSampleQuery(
                    sampleType: standType,
                    predicate: predicate,
                    limit: HKObjectQueryNoLimit,
                    sortDescriptors: nil
                ) { query, samples, error in
                    if let error = error {
                        continuation.resume(throwing: error)
                        return
                    }

                    // Count stood hours (value == .stood)
                    let stoodCount = (samples as? [HKCategorySample])?.filter { $0.value == HKCategoryValueAppleStandHour.stood.rawValue }.count ?? 0
                    continuation.resume(returning: stoodCount)
                }

                self.healthStore.execute(query)
            }

            if count > 0 {
                let dateString = formatter.string(from: currentDate)
                dailyStand[dateString] = count
            }

            currentDate = nextDate
        }

        return dailyStand
    }

    // MARK: - Sleep

    func fetchSleepData(from: Date, to: Date) async throws -> (daily: [String: (totalMin: Int, stages: SleepStages?)], segments: [SleepSegment]) {
        guard let sleepType = HKObjectType.categoryType(forIdentifier: .sleepAnalysis) else {
            return (daily: [:], segments: [])
        }

        let calendar = Calendar.current
        let startOfDay = calendar.startOfDay(for: from)
        let endOfDay = calendar.date(byAdding: .day, value: 1, to: calendar.startOfDay(for: to))!

        let predicate = HKQuery.predicateForSamples(withStart: startOfDay, end: endOfDay, options: .strictStartDate)
        let sortDescriptor = NSSortDescriptor(key: HKSampleSortIdentifierStartDate, ascending: true)

        let samples = try await withCheckedThrowingContinuation { (continuation: CheckedContinuation<[HKCategorySample], Error>) in
            let query = HKSampleQuery(
                sampleType: sleepType,
                predicate: predicate,
                limit: HKObjectQueryNoLimit,
                sortDescriptors: [sortDescriptor]
            ) { query, samples, error in
                if let error = error {
                    continuation.resume(throwing: error)
                    return
                }
                continuation.resume(returning: (samples as? [HKCategorySample]) ?? [])
            }

            healthStore.execute(query)
        }

        var segments: [SleepSegment] = []
        var dailyTotals: [String: [String: Int]] = [:] // date -> stage -> minutes
        let formatter = DateFormatter()
        formatter.dateFormat = "yyyy-MM-dd"

        for sample in samples {
            let duration = sample.endDate.timeIntervalSince(sample.startDate)
            let minutes = Int(duration / 60)

            // Map HKCategoryValueSleepAnalysis to stage
            let stage: String
            if #available(iOS 16.0, *) {
                switch sample.value {
                case HKCategoryValueSleepAnalysis.asleepREM.rawValue:
                    stage = "rem"
                case HKCategoryValueSleepAnalysis.asleepDeep.rawValue:
                    stage = "deep"
                case HKCategoryValueSleepAnalysis.asleepCore.rawValue:
                    stage = "core"
                case HKCategoryValueSleepAnalysis.awake.rawValue:
                    stage = "awake"
                case HKCategoryValueSleepAnalysis.asleepUnspecified.rawValue:
                    stage = "core" // fallback
                default:
                    stage = "core"
                }
            } else {
                // iOS 15 and earlier - simpler mapping
                switch sample.value {
                case HKCategoryValueSleepAnalysis.awake.rawValue:
                    stage = "awake"
                default:
                    stage = "core" // treat all sleep as core
                }
            }

            segments.append(SleepSegment(start: sample.startDate, end: sample.endDate, stage: stage))

            // Aggregate by date (use start date)
            let dateString = formatter.string(from: sample.startDate)
            if dailyTotals[dateString] == nil {
                dailyTotals[dateString] = [:]
            }
            dailyTotals[dateString]![stage, default: 0] += minutes
        }

        // Build daily summary
        var dailySummary: [String: (totalMin: Int, stages: SleepStages?)] = [:]
        for (date, stages) in dailyTotals {
            let rem = stages["rem"] ?? 0
            let deep = stages["deep"] ?? 0
            let core = stages["core"] ?? 0
            let awake = stages["awake"] ?? 0
            let total = rem + deep + core + awake

            if total > 0 {
                let stagesData = SleepStages(rem: rem, deep: deep, core: core, awake: awake)
                dailySummary[date] = (totalMin: total, stages: stagesData)
            }
        }

        return (daily: dailySummary, segments: segments)
    }

    // MARK: - Workouts

    func fetchWorkouts(from: Date, to: Date) async throws -> [WorkoutSession] {
        let calendar = Calendar.current
        let startOfDay = calendar.startOfDay(for: from)
        let endOfDay = calendar.date(byAdding: .day, value: 1, to: calendar.startOfDay(for: to))!

        let predicate = HKQuery.predicateForSamples(withStart: startOfDay, end: endOfDay, options: .strictStartDate)
        let sortDescriptor = NSSortDescriptor(key: HKSampleSortIdentifierStartDate, ascending: true)

        return try await withCheckedThrowingContinuation { (continuation: CheckedContinuation<[WorkoutSession], Error>) in
            let query = HKSampleQuery(
                sampleType: HKObjectType.workoutType(),
                predicate: predicate,
                limit: HKObjectQueryNoLimit,
                sortDescriptors: [sortDescriptor]
            ) { query, samples, error in
                if let error = error {
                    continuation.resume(throwing: error)
                    return
                }

                let workouts = (samples as? [HKWorkout])?.map { workout -> WorkoutSession in
                    let label = self.mapWorkoutType(workout.workoutActivityType)
                    let calories = workout.totalEnergyBurned.map { Int($0.doubleValue(for: .kilocalorie())) }

                    return WorkoutSession(
                        start: workout.startDate,
                        end: workout.endDate,
                        label: label,
                        caloriesKcal: calories
                    )
                } ?? []

                continuation.resume(returning: workouts)
            }

            healthStore.execute(query)
        }
    }

    private func mapWorkoutType(_ type: HKWorkoutActivityType) -> String {
        switch type {
        case .running: return "run"
        case .walking: return "walk"
        case .traditionalStrengthTraining: return "strength"
        case .coreTraining: return "core"
        case .cycling: return "cycle"
        case .swimming: return "swim"
        case .yoga: return "yoga"
        case .hiking: return "hike"
        default: return "other"
        }
    }

    // MARK: - Wrist Temperature

    func fetchDailyWristTemp(from: Date, to: Date) async throws -> [String: (avg: Double, min: Double?, max: Double?)] {
        guard let tempType = HKQuantityType.quantityType(forIdentifier: .appleSleepingWristTemperature) else {
            return [:] // Not available on this device
        }

        let calendar = Calendar.current
        let startOfDay = calendar.startOfDay(for: from)
        let endOfDay = calendar.date(byAdding: .day, value: 1, to: calendar.startOfDay(for: to))!

        let predicate = HKQuery.predicateForSamples(withStart: startOfDay, end: endOfDay, options: .strictStartDate)

        var anchorComponents = calendar.dateComponents([.year, .month, .day], from: startOfDay)
        anchorComponents.hour = 0
        let anchorDate = calendar.date(from: anchorComponents)!

        let interval = DateComponents(day: 1)

        return try await withCheckedThrowingContinuation { continuation in
            let query = HKStatisticsCollectionQuery(
                quantityType: tempType,
                quantitySamplePredicate: predicate,
                options: [.discreteAverage, .discreteMin, .discreteMax],
                anchorDate: anchorDate,
                intervalComponents: interval
            )

            query.initialResultsHandler = { query, results, error in
                if let error = error {
                    continuation.resume(throwing: error)
                    return
                }

                guard let results = results else {
                    continuation.resume(returning: [:])
                    return
                }

                var dailyTemp: [String: (avg: Double, min: Double?, max: Double?)] = [:]
                let formatter = DateFormatter()
                formatter.dateFormat = "yyyy-MM-dd"

                results.enumerateStatistics(from: startOfDay, to: endOfDay) { statistics, stop in
                    if let avg = statistics.averageQuantity() {
                        let avgC = avg.doubleValue(for: .degreeCelsius())
                        let minC = statistics.minimumQuantity()?.doubleValue(for: .degreeCelsius())
                        let maxC = statistics.maximumQuantity()?.doubleValue(for: .degreeCelsius())

                        let dateString = formatter.string(from: statistics.startDate)
                        dailyTemp[dateString] = (avg: avgC, min: minC, max: maxC)
                    }
                }

                continuation.resume(returning: dailyTemp)
            }

            healthStore.execute(query)
        }
    }

    // MARK: - Daily Weight Last

    func fetchDailyWeightLast(from: Date, to: Date) async throws -> [String: Double] {
        let weightType = HKQuantityType.quantityType(forIdentifier: .bodyMass)!

        let calendar = Calendar.current
        let startOfDay = calendar.startOfDay(for: from)
        let endOfDay = calendar.date(byAdding: .day, value: 1, to: calendar.startOfDay(for: to))!

        var dailyWeights: [String: Double] = [:]
        let formatter = DateFormatter()
        formatter.dateFormat = "yyyy-MM-dd"

        // Iterate through each day
        var currentDate = startOfDay
        while currentDate < endOfDay {
            let nextDate = calendar.date(byAdding: .day, value: 1, to: currentDate)!
            let predicate = HKQuery.predicateForSamples(withStart: currentDate, end: nextDate, options: .strictStartDate)

            let sortDescriptor = NSSortDescriptor(key: HKSampleSortIdentifierEndDate, ascending: false)

            let weight = try await withCheckedThrowingContinuation { (continuation: CheckedContinuation<Double?, Error>) in
                let query = HKSampleQuery(
                    sampleType: weightType,
                    predicate: predicate,
                    limit: 1,
                    sortDescriptors: [sortDescriptor]
                ) { query, samples, error in
                    if let error = error {
                        continuation.resume(throwing: error)
                        return
                    }

                    if let sample = samples?.first as? HKQuantitySample {
                        let weightKg = sample.quantity.doubleValue(for: HKUnit.gramUnit(with: .kilo))
                        continuation.resume(returning: weightKg)
                    } else {
                        continuation.resume(returning: nil)
                    }
                }

                self.healthStore.execute(query)
            }

            if let weight = weight {
                let dateString = formatter.string(from: currentDate)
                dailyWeights[dateString] = weight
            }

            currentDate = nextDate
        }

        return dailyWeights
    }

    // MARK: - Build Sync Request

    func buildSyncRequest(profileId: UUID, from: Date, to: Date) async throws -> SyncBatchRequest {
        let calendar = Calendar.current
        let formatter = DateFormatter()
        formatter.dateFormat = "yyyy-MM-dd"

        // Fetch all data concurrently
        async let dailyStepsData = try fetchDailySteps(from: from, to: to)
        async let dailyRestingHRData = try fetchDailyRestingHR(from: from, to: to)
        async let dailyWeightsData = try fetchDailyWeightLast(from: from, to: to)
        async let dailyNutritionData = try fetchDailyNutrition(from: from, to: to)
        async let dailyActivityData = try fetchDailyActivity(from: from, to: to)
        async let sleepData = try fetchSleepData(from: from, to: to)
        async let workoutsData = try fetchWorkouts(from: from, to: to)
        async let dailyTempData = try fetchDailyWristTemp(from: from, to: to)

        let stepsMap = try await Dictionary(uniqueKeysWithValues: dailyStepsData)
        let restingHRMap = try await dailyRestingHRData
        let weightsMap = try await dailyWeightsData
        let nutritionMap = try await dailyNutritionData
        let activityMap = try await dailyActivityData
        let (sleepDaily, sleepSegments) = try await sleepData
        let workouts = try await workoutsData
        let tempMap = try await dailyTempData

        // Build daily aggregates
        var dailyAggregates: [DailyAggregate] = []
        var currentDate = calendar.startOfDay(for: from)
        let endDate = calendar.startOfDay(for: to)

        while currentDate <= endDate {
            let dateString = formatter.string(from: currentDate)

            // Activity
            var activity: ActivityDaily? = nil
            let steps = stepsMap[dateString] ?? 0
            if let actData = activityMap[dateString] {
                activity = ActivityDaily(
                    steps: steps,
                    activeEnergyKcal: actData.activeEnergy,
                    exerciseMin: actData.exerciseMin,
                    standHours: actData.standHours,
                    distanceKm: actData.distance
                )
            } else if steps > 0 {
                // Have steps but no other activity data
                activity = ActivityDaily(
                    steps: steps,
                    activeEnergyKcal: 0,
                    exerciseMin: 0,
                    standHours: 0,
                    distanceKm: 0
                )
            }

            // Heart
            var heart: HeartDaily? = nil
            if let restingHR = restingHRMap[dateString] {
                heart = HeartDaily(restingHrBpm: restingHR)
            }

            // Body
            var body: BodyDaily? = nil
            if let weight = weightsMap[dateString] {
                body = BodyDaily(
                    weightKgLast: weight,
                    bmi: 0,
                    bodyFatPct: nil
                )
            }

            // Sleep
            var sleep: SleepDaily? = nil
            if let sleepData = sleepDaily[dateString] {
                sleep = SleepDaily(
                    totalMinutes: sleepData.totalMin,
                    stages: sleepData.stages
                )
            }

            // Nutrition
            let nutrition = nutritionMap[dateString]

            // Temperature
            var temperature: TemperatureDaily? = nil
            if let temp = tempMap[dateString] {
                temperature = TemperatureDaily(
                    wristCAvg: temp.avg,
                    wristCMin: temp.min,
                    wristCMax: temp.max
                )
            }

            // Only add if we have any data
            if activity != nil || heart != nil || body != nil || sleep != nil || nutrition != nil || temperature != nil {
                dailyAggregates.append(DailyAggregate(
                    date: dateString,
                    sleep: sleep,
                    activity: activity,
                    body: body,
                    heart: heart,
                    nutrition: nutrition,
                    intakes: nil,
                    temperature: temperature
                ))
            }

            currentDate = calendar.date(byAdding: .day, value: 1, to: currentDate)!
        }

        // Fetch hourly data for each day
        var allHourlyBuckets: [HourlyBucket] = []
        currentDate = calendar.startOfDay(for: from)

        while currentDate <= endDate {
            async let hourlyStepsData = try fetchHourlySteps(for: currentDate)
            async let hourlyHRData = try fetchHourlyHeartRate(for: currentDate)

            let stepsBuckets = try await hourlyStepsData
            let hrBuckets = try await hourlyHRData

            // Merge steps and HR buckets
            var bucketsByHour: [Date: HourlyBucket] = [:]

            for bucket in stepsBuckets {
                bucketsByHour[bucket.hour] = bucket
            }

            for bucket in hrBuckets {
                if let existing = bucketsByHour[bucket.hour] {
                    // Merge: create new bucket with both steps and hr
                    let merged = HourlyBucket(hour: existing.hour, steps: existing.steps, hr: bucket.hr)
                    bucketsByHour[bucket.hour] = merged
                } else {
                    bucketsByHour[bucket.hour] = bucket
                }
            }

            allHourlyBuckets.append(contentsOf: bucketsByHour.values.sorted(by: { $0.hour < $1.hour }))

            currentDate = calendar.date(byAdding: .day, value: 1, to: currentDate)!
        }

        return SyncBatchRequest(
            profileId: profileId,
            clientTimeZone: TimeZone.current.identifier,
            daily: dailyAggregates.isEmpty ? nil : dailyAggregates,
            hourly: allHourlyBuckets.isEmpty ? nil : allHourlyBuckets,
            sessions: Sessions(
                sleepSegments: sleepSegments.isEmpty ? nil : sleepSegments,
                workouts: workouts.isEmpty ? nil : workouts
            )
        )
    }
}

// MARK: - Errors

enum HealthKitError: LocalizedError {
    case notAvailable
    case authorizationFailed
    case dataNotAvailable
    case typeNotAvailable

    var errorDescription: String? {
        switch self {
        case .notAvailable:
            return "HealthKit недоступен на этом устройстве"
        case .authorizationFailed:
            return "Не удалось получить разрешение на доступ к данным здоровья"
        case .dataNotAvailable:
            return "Данные недоступны"
        case .typeNotAvailable:
            return "Тип данных не поддерживается HealthKit"
        }
    }
}
