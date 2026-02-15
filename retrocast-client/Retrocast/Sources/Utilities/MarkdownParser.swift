import Foundation
import SwiftUI

/// Basic inline markdown parsing: **bold**, *italic*, `code`, ~~strikethrough~~.
enum MarkdownParser {
    static func parse(_ text: String) -> AttributedString {
        // Use SwiftUI's built-in markdown support for basic formatting
        if let attributed = try? AttributedString(markdown: text,
                                                   options: .init(interpretedSyntax: .inlineOnlyPreservingWhitespace)) {
            return attributed
        }
        return AttributedString(text)
    }
}
