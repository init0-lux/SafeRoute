import React from 'react';
import { View, StyleSheet } from 'react-native';
import { Colors } from '@/constants/theme';

interface PaginationDotsProps {
    activeIndex: number;
    count?: number;
    colors: string[];
}

export default function PaginationDots({
    activeIndex,
    count = 3,
    colors,
}: PaginationDotsProps) {
    return (
        <View style={styles.container}>
            {Array.from({ length: count }).map((_, index) => (
                <View
                    key={index}
                    style={[
                        styles.dot,
                        {
                            backgroundColor:
                                index === activeIndex
                                    ? colors[index] || Colors.white
                                    : Colors.grayDark,
                            width: index === activeIndex ? 10 : 8,
                            height: index === activeIndex ? 10 : 8,
                        },
                    ]}
                />
            ))}
        </View>
    );
}

const styles = StyleSheet.create({
    container: {
        flexDirection: 'row',
        justifyContent: 'center',
        alignItems: 'center',
        gap: 8,
    },
    dot: {
        borderRadius: 9999,
    },
});
