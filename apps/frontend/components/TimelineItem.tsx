import React from 'react';
import { View, Text, StyleSheet } from 'react-native';
import { Ionicons } from '@expo/vector-icons';
import { Colors, Spacing, BorderRadius, FontSizes } from '@/constants/theme';

interface TimelineItemProps {
    title: string;
    date: string;
    description: string;
    icon?: keyof typeof Ionicons.glyphMap;
    titleColor?: string;
    isLast?: boolean;
}

export default function TimelineItem({
    title,
    date,
    description,
    icon = 'time-outline',
    titleColor = Colors.red,
    isLast = false,
}: TimelineItemProps) {
    return (
        <View style={styles.container}>
            {/* Left: icon + line */}
            <View style={styles.leftColumn}>
                <View style={styles.iconCircle}>
                    <Ionicons name={icon} size={18} color={Colors.purple} />
                </View>
                {!isLast && <View style={styles.line} />}
            </View>

            {/* Right: content card */}
            <View style={styles.card}>
                <Text style={[styles.title, { color: titleColor }]}>{title}</Text>
                <Text style={styles.date}>{date}</Text>
                <Text style={styles.description}>{description}</Text>
            </View>
        </View>
    );
}

const styles = StyleSheet.create({
    container: {
        flexDirection: 'row',
        minHeight: 100,
    },
    leftColumn: {
        width: 40,
        alignItems: 'center',
    },
    iconCircle: {
        width: 36,
        height: 36,
        borderRadius: 18,
        backgroundColor: Colors.cardBgLight,
        justifyContent: 'center',
        alignItems: 'center',
        borderWidth: 1,
        borderColor: Colors.border,
    },
    line: {
        flex: 1,
        width: 2,
        backgroundColor: Colors.border,
        marginVertical: 4,
    },
    card: {
        flex: 1,
        backgroundColor: Colors.cardBg,
        borderRadius: BorderRadius.md,
        padding: Spacing.md,
        marginLeft: Spacing.md,
        marginBottom: Spacing.md,
        borderWidth: 1,
        borderColor: Colors.border,
    },
    title: {
        fontSize: FontSizes.sm,
        fontWeight: '700',
        textTransform: 'uppercase',
        letterSpacing: 0.5,
        marginBottom: 2,
    },
    date: {
        fontSize: FontSizes.xs,
        color: Colors.gray,
        marginBottom: Spacing.sm,
    },
    description: {
        fontSize: FontSizes.sm,
        color: Colors.grayLight,
        lineHeight: FontSizes.sm * 1.5,
    },
});
