import SwiftUI

struct DateSeparatorView: View {
    let date: Date

    var body: some View {
        HStack {
            line
            Text(DateFormatting.dateSeparator(date))
                .font(.caption)
                .fontWeight(.semibold)
                .foregroundStyle(.retroMuted)
                .fixedSize()
            line
        }
        .padding(.horizontal, 16)
        .padding(.vertical, 8)
    }

    private var line: some View {
        VStack { Divider() }
    }
}
